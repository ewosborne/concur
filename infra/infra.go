package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
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
	Stderr           []string      `json:"stderr"`
	StartTime        time.Time     `json:"starttime"`
	EndTime          time.Time     `json:"endtime"`
	RunTimePrintable string        `json:"runtime"`
	RunTime          time.Duration `json:"-"` // msec runtime for sorting
	ReturnCode       int           `json:"returncode"`
}

type Flags struct {
	Any             bool
	ConcurrentLimit string
	GoroutineLimit  int // derived from ConcurrencyLimit
	Timeout         time.Duration
	Token           string
	FlagErrors      bool
	FirstZero       bool
	Pbar            bool
}

type CommandList []*Command

// type CommandMap map[JobID]*Command
type CommandMap map[int]*Command

func (c Command) String() string {
	b, _ := json.MarshalIndent(c, "", " ")
	return string(b)

}

var flagErrors bool

func Do(command string, substituteArgs []string, flags Flags) Results {
	// do all the heavy lifting here
	var ctx context.Context
	var cancelCtx context.CancelFunc
	var res = Results{}

	flagErrors = flags.FlagErrors
	systemStartTime := time.Now()

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
		fmt.Fprint(os.Stderr, err)
	}

	// flag fixup.
	// need this here because PopulateFlags doesn't get cmdList.
	// TODO this is messy and in need of cleanup
	if flags.GoroutineLimit == 0 {
		flags.GoroutineLimit = len(cmdList)
	}
	// go run the things
	completedCommands, pbarOffset := command_loop(ctx, cmdList, flags)

	// finalizing
	systemEndTime := time.Now()
	systemRunTime := systemEndTime.Sub(systemStartTime)
	systemRunTime = systemRunTime - pbarOffset

	// return Results map
	res.Commands = completedCommands
	res.Info.InternalSystemRunTime = systemRunTime
	res.Info.CoroutineLimit = flags.GoroutineLimit

	return res
}

type Results struct {
	//Commands CommandMap  `json:"commands"`
	Commands CommandList `json:"command"`

	Info ResultsInfo `json:"info"`
}

type ResultsInfo struct {
	CoroutineLimit        int
	InternalSystemRunTime time.Duration `json:"-"`
	SystemRuntime         string
}

func GetJSONReport(res Results) (string, error) {
	res.Info.SystemRuntime = res.Info.InternalSystemRunTime.Truncate(time.Millisecond).String()
	jsonResults, err := json.MarshalIndent(res, "", " ")
	if err != nil {
		slog.Error("error marshaling results")
		return "", err
	}

	return string(jsonResults), nil
}

func ReportDone(res Results, flags Flags) {

	jsonResults, err := GetJSONReport(res)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
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

func executeSingleCommand(ctx context.Context, c *Command) {

	var outb, errb strings.Builder

	// name is command name, args is slice of arguments to that command
	f := strings.Fields(c.Substituted)
	name, args := f[0], f[1:]

	cmd := exec.CommandContext(ctx, name, args...)

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

func get_pbar(cmdList CommandList, flags Flags) *progressbar.ProgressBar {
	pbar := progressbar.NewOptions(len(cmdList),
		progressbar.OptionSetVisibility(flags.Pbar),
		progressbar.OptionSetItsString("jobs"), // doesn't do anything, don't know why
		progressbar.OptionShowCount(),
		progressbar.OptionSetWriter(os.Stderr),
	)

	pbar.RenderBlank() // to get it to render at 0% before any job finishes

	return pbar

}

func command_loop(ctx context.Context, cmdList CommandList, flags Flags) (CommandList, time.Duration) {

	var tokens = make(chan struct{}, flags.GoroutineLimit) // permission to run
	var done = make(chan *Command)                         // where a command goes when it's done
	var completedCommands CommandList                      // count all the done processes
	var pbarFinish time.Duration
	var completionCount int

	// small fixed delay after printing the end of the pbar so we can see that it hit 100%
	if flags.Pbar {
		pbarFinish = time.Duration(250 * time.Millisecond)
	}

	// a jobcount pbar, doesn't print anything unless flags.Pbar is set
	pbar := get_pbar(cmdList, flags)

	// launch all goroutines

	for _, c := range cmdList {

		go func() {
			tokens <- struct{}{} // get permission to start
			c.StartTime = time.Now()

			executeSingleCommand(ctx, c)
			c.EndTime = time.Now()
			c.RunTime = c.EndTime.Sub(c.StartTime)
			c.RunTimePrintable = c.RunTime.String()

			done <- c // report status.
			<-tokens  // return token when done.
		}()

	}

	// collect all goroutines

	doneList := CommandList{}
Outer:
	for completionCount != len(cmdList) {

		select {
		case c := <-done:
			completionCount += 1
			doneList = append(doneList, c)
			//fmt.Println("appended", c.RunTime)

			// this is for the job-based progressbar
			// TODO: display substituted hostname in completion pbar?
			pbar.Add(1)
			if flags.FirstZero || (flags.Any && c.ReturnCode == 0) {
				slog.Debug(fmt.Sprintf("returning %s", c.Arg))
				// this only returns the single command we're interested in regardless of what other commands have done.
				//  TODO is this what I want?  or do I want to return all commands but the other ones as NotStarted?
				break Outer
			}

		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "context popped, %v jobs done", len(completedCommands))
			break Outer
		}
	}
	// Outer: breaks here

	// sort doneList by completion time so .commands[0] is the fastest.
	sort.Slice(doneList, func(i int, j int) bool {
		return doneList[i].RunTime < doneList[j].RunTime
	})

	pbar.Finish()          // don't know if I need this.
	time.Sleep(pbarFinish) // to let the pbar finish displaying.

	return doneList, pbarFinish
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
			fmt.Fprintf(os.Stderr, "Invalid concurrency level: %s\n", flags.ConcurrentLimit)
			os.Exit(1)
		}
		flags.GoroutineLimit = x
	}

	tmp, _ := cmd.Flags().GetInt64("timeout")
	flags.Timeout = time.Duration(tmp) * time.Second

	flags.Token, _ = cmd.Flags().GetString("token")
	flags.FlagErrors, _ = cmd.Flags().GetBool("flag-errors")
	flags.FirstZero, _ = cmd.Flags().GetBool("first")
	flags.Pbar, _ = cmd.Flags().GetBool("pbar")
	return flags
}

func buildListOfCommands(command string, hosts []string, token string) (CommandList, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.
	// TODO rename hosts and host to something less specific

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

// return whether there's something to read on stdin
func GetStdin() ([]string, bool) {

	fi, _ := os.Stdin.Stat()

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		//fmt.Println("reading from stdin")

		bytes, _ := io.ReadAll(os.Stdin)
		str := string(bytes)
		//fmt.Println(str, len(str))

		// TODO: think about this.  would I ever want
		//  a multi-word arg?
		return strings.Fields(str), true
	} else {
		//fmt.Println("reading from terminal")
	}
	return nil, false
}
