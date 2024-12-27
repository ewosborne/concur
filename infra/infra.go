package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type JobStatus int
type JobID int

var id JobID

const (
	TBD JobStatus = iota
	Started
	Running
	Finished
	Errored
)

func (j JobStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.String())
}

func (j JobStatus) String() string {
	switch j {
	case TBD:
		return "TBD"
	case Started:
		return "Started"
	case Running:
		return "Running"
	case Finished:
		return "Finished"
	case Errored:
		return "Errored"
	default:
		return "Unknown"
	}
}

type Command struct {
	ID          JobID     `json:"id"`
	Status      JobStatus `json:"jobstatus"`
	Original    string    `json:"original"`
	Substituted string    `json:"substituted"`
	Arg         string    `json:"arg"`
	Stdout      []string  `json:"stdout"`
	//Stdin       string    `json:"stdin"`
	Stderr     []string  `json:"stderr"`
	StartTime  time.Time `json:"starttime"`
	EndTime    time.Time `json:"endtime"`
	RunTime    string    `json:"runtime"`
	ReturnCode int       `json:"returncode"`
}

type Flags struct {
	Any             bool
	ConcurrentLimit string
	GoroutineLimit  int // derived from ConcurrencyLimit
	Timeout         time.Duration
	Token           string
	FlagErrors      bool
	FirstZero       bool
}

// for now, sometimes passed into things
//
//	TODO do away with this and just use CommandMap and range over it?
type CommandList []*Command

type CommandMap map[JobID]*Command

func (c Command) String() string {
	b, _ := json.MarshalIndent(c, "", " ") // TODO clean this up, particularly RunTime
	return string(b)

}

var flagErrors bool

func Do(command string, substituteArgs []string, flags Flags) Results {
	// do all the heavy lifting here
	var ctx context.Context
	var cancelCtx context.CancelFunc

	flagErrors = flags.FlagErrors
	systemStartTime := time.Now()

	// TODO: pass flags in as float in seconds, convert to integer msec

	switch flags.Timeout {
	case 0:
		ctx, cancelCtx = context.WithCancel(context.Background())

	default:
		ctx, cancelCtx = context.WithTimeout(context.Background(), flags.Timeout)
	}

	defer cancelCtx()

	// build a list of commands
	cmdList, err := buildListOfCommands(command, substituteArgs, flags.Token)
	if err != nil {
		panic(err) // TODO fix
	}

	// flag fixup
	if flags.GoroutineLimit == 0 {
		flags.GoroutineLimit = len(cmdList)
	}
	// go run the things
	completedCommands := command_loop(ctx, cmdList, flags)

	// finalizing
	systemEndTime := time.Now()
	systemRunTime := systemEndTime.Sub(systemStartTime)

	// return Results map
	var res = Results{}

	res.Commands = completedCommands
	res.Info.InternalSystemRunTime = systemRunTime
	res.Info.CoroutineLimit = flags.GoroutineLimit

	return res

	//reportDone(completedCommands, systemRunTime, flags)
}

type Results struct {
	Commands CommandMap  `json:"commands"`
	Info     ResultsInfo `json:"info"`
}

type ResultsInfo struct {
	CoroutineLimit        int
	InternalSystemRunTime time.Duration `json:"-"`
	SystemRuntime         string
}

func GetJSONReport(res Results) (string, error) {
	// TODO how do I display runtime as a string?
	//res.Info.InternalSystemRunTime = res.Info.InternalSystemRunTime / time.Millisecond
	res.Info.SystemRuntime = res.Info.InternalSystemRunTime.Truncate(time.Millisecond).String()
	jsonResults, err := json.MarshalIndent(res, "", " ")
	if err != nil {
		slog.Error("error marshaling results")
		return "", err
	}

	return string(jsonResults), nil
}

func ReportDone(res Results) {

	// little fixup for printout but still fucky and I don't like it.

	jsonResults, err := GetJSONReport(res)
	if err != nil {
		panic("todo fixme")
	}
	fmt.Println(jsonResults)

	if flagErrors {
		for _, c := range res.Commands {
			if c.ReturnCode != 0 {
				// TODO better format?
				fmt.Fprintf(os.Stderr, "command %v exited with error code %v\n", c.Substituted, c.ReturnCode)
			}
		}
	}

}

func execute(ctx context.Context, c *Command) {

	f := strings.Fields(c.Substituted)
	name, args := f[0], f[1:]

	cmd := exec.CommandContext(ctx, name, args...)

	var outb, errb strings.Builder
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	c.Status = Running
	err := cmd.Run()

	if err != nil {
		c.Status = Errored
		if exitError, ok := err.(*exec.ExitError); ok {
			c.ReturnCode = exitError.ExitCode()
		}
	} else {
		c.Status = Finished
		c.ReturnCode = 0
	}

	c.Stdout = strings.Split(outb.String(), "\n")
	c.Stderr = strings.Split(errb.String(), "\n")

}

func command_loop(ctx context.Context, cmdList CommandList, flags Flags) CommandMap {

	// TODO if I'm going to get clever about concurrnetLimit being a string, maybe it's here?
	var tokens = make(chan struct{}, flags.GoroutineLimit) // permission to run
	var done = make(chan *Command)                         // where a command goes when it's done
	var completedCommands CommandList                      // count all the done processes
	var cmdMap = CommandMap{}

	// launch all goroutines
	for _, c := range cmdList {
		cmdMap[c.ID] = c
		go func() {
			tokens <- struct{}{} // get permission to start
			c.StartTime = time.Now()

			execute(ctx, c)
			c.EndTime = time.Now()
			rt := c.EndTime.Sub(c.StartTime)
			a, err := time.ParseDuration(rt.String())
			if err != nil {
				panic(err) // TODO
			}
			c.RunTime = a.String()

			done <- c // report status.
			<-tokens  // return token when done.
		}()

	}

	// TODO should break this loop out into another function I guess.
	// TODO should completionCount be a waitgroup? does it matter? maybe or maybe not

	// gather everything we want and then return
	var completionCount int

Outer:
	for completionCount != len(cmdMap) {

		select {
		case c := <-done:
			completionCount += 1

			if flags.FirstZero || (flags.Any && c.ReturnCode == 0) {
				slog.Debug(fmt.Sprintf("returning %s", c.Arg))
				// this only returns the single command we're interested in.  TODO is that what I want?
				cmdMap = CommandMap{c.ID: c}
				break Outer
			}

		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "context popped, %v jobs done", len(completedCommands))
			break Outer
		}
	}
	// Outer: breaks here
	return cmdMap
}

func PopulateFlags(cmd *cobra.Command) Flags {
	flags := Flags{}
	// I sure wish there was a cleaner way to do this
	flags.Any, _ = cmd.Flags().GetBool("any")
	flags.ConcurrentLimit, _ = cmd.Flags().GetString("concurrent")

	switch flags.ConcurrentLimit {
	case "cpu", "1x":
		flags.GoroutineLimit = runtime.NumCPU()
	case "2x":
		flags.GoroutineLimit = runtime.NumCPU() * 2
	default:
		x, err := strconv.Atoi(flags.ConcurrentLimit)
		if err != nil {
			panic("bad flag number")
		}
		flags.GoroutineLimit = x
	}

	tmp, _ := cmd.Flags().GetInt64("timeout")
	flags.Timeout = time.Duration(tmp) * time.Second

	flags.Token, _ = cmd.Flags().GetString("token")
	flags.FlagErrors, _ = cmd.Flags().GetBool("flag-errors")
	flags.FirstZero, _ = cmd.Flags().GetBool("first")
	return flags
}

func buildListOfCommands(command string, hosts []string, token string) (CommandList, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.

	var ret CommandList
	for _, host := range hosts {
		x := Command{}
		x.Original = command
		x.Arg = host
		x.Substituted = strings.ReplaceAll(command, token, host)
		x.Status = TBD
		x.ID = id

		id += 1

		ret = append(ret, &x)
	}

	// mix them up just so there's no ordering dependency if they all take about the same time. otherwise the first one in the list
	//   tends to be the one we return first with --any.
	rand.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})

	return ret, nil
}
