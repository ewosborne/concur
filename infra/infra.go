package infra

// TODO reorder this whole file.
import (
	"context"
	"encoding/json"
	"fmt"
	_ "log" // magic to make slog look like log
	"log/slog"
	"math"
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

const (
	TBD JobStatus = iota
	Started
	Running
	Finished
	Errored
	TimedOut
)

var flagErrors bool

type JobID int
type JobStatus int

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
	case TimedOut:
		return "TimedOut"
	default:
		return "Unknown"
	}
}

type Results struct {
	Commands CommandList `json:"command"`
	Info     ResultsInfo `json:"info"`
}

type ResultsInfo struct {
	CoroutineLimit        int           `json:"coroutineLimit"`
	InternalSystemRunTime time.Duration `json:"-"`
	SystemRuntimeString   string        `json:"systemRuntime"`
	OriginalCommand       string        `json:"originalCommand"`
	Timeout               time.Duration `json:"timeout"` // rename this?
}

type Command struct {
	ID          JobID     `json:"id"`
	Status      JobStatus `json:"jobstatus"`
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
	JobTimeout       time.Duration `json:"jobtimeout"` // TODO these print as ints, would be nice to print as string.
}

func (c Command) String() string {
	b, _ := json.MarshalIndent(c, "", " ")
	return string(b)

}

type CommandList []*Command

// type CommandMap map[JobID]*Command
type CommandMap map[int]*Command

// TODO I think most of my testing is around varying these flags.
type Flags struct {
	Any                bool
	ConcurrentJobLimit string
	GoroutineLimit     int // derived from ConcurrentJobLimit
	Timeout            time.Duration
	Token              string
	FlagErrors         bool
	FirstZero          bool
	Pbar               bool
	JobTimeout         time.Duration
	LogLevel           string
}

func Do(template string, targets []string, flags Flags) Results {
	// do all the heavy lifting here
	var ctx context.Context
	var cancelCtx context.CancelFunc
	var res = Results{}

	slog.Debug(fmt.Sprintf("calling Do with %v %v %v", template, targets, flags))

	flagErrors = flags.FlagErrors
	systemStartTime := time.Now()

	switch flags.Timeout {
	case 0:
		ctx, cancelCtx = context.WithCancel(context.Background())

	default:
		ctx, cancelCtx = context.WithTimeout(context.Background(), flags.Timeout)
	}

	//ctx = loginfra.WithLogger(ctx, Logger)
	defer cancelCtx()

	// build a list of commandsToRun
	commandsToRun, err := buildListOfCommands(template, targets, flags.Token)
	if err != nil {
		//fmt.Fprint(os.Stderr, err)
		slog.Error(fmt.Sprintf("error building list of commands: %v", err))
	}

	// flag fixup.
	// need this here because PopulateFlags doesn't get cmdList.
	// TODO this is messy and in need of cleanup
	if flags.GoroutineLimit == 0 {
		flags.GoroutineLimit = len(commandsToRun)
	}
	// go run the things
	completedCommands, pbarOffset := commandLoop(ctx, cancelCtx, commandsToRun, flags)

	// finalizing
	systemEndTime := time.Now()
	systemRunTime := systemEndTime.Sub(systemStartTime)
	systemRunTime = systemRunTime - pbarOffset

	// return Results map
	res.Commands = completedCommands
	res.Info.InternalSystemRunTime = systemRunTime
	res.Info.CoroutineLimit = flags.GoroutineLimit
	res.Info.OriginalCommand = template
	res.Info.Timeout = flags.Timeout

	return res
}

func GetJSONReport(res Results) (string, error) {
	res.Info.SystemRuntimeString = res.Info.InternalSystemRunTime.Truncate(time.Millisecond).String()
	jsonResults, err := json.MarshalIndent(res, "", " ")
	if err != nil {
		// TODO  slog.Error("error marshaling results")
		return "", err
	}

	return string(jsonResults), nil
}

func ReportDone(res Results, flags Flags) {

	jsonResults, err := GetJSONReport(res)
	if err != nil {
		//fmt.Fprint(os.Stderr, err)
		slog.Error(fmt.Sprintf("error getting json report: %v", err))
	}
	fmt.Println(jsonResults)

	if flagErrors {
		for _, c := range res.Commands {
			if c.ReturnCode != 0 {
				// TODO better format?
				//fmt.Fprintf(os.Stderr, "command %v exited with error code %v\n", c.Substituted, c.ReturnCode)
				slog.Error(fmt.Sprintf("command %v exited with error code %v\n", c.Substituted, c.ReturnCode))
			}
		}
	}

}

// TODO return an error here?  who'd receive it?
func executeSingleCommand(jobCtx context.Context, jobCancel context.CancelFunc, c *Command) {

	var outb, errb strings.Builder

	//l := loginfra.GetLogger(jobCtx)
	//l.Warning("esc warn")

	defer jobCancel() // I assume I need this - ??

	// name is command name, args is slice of arguments to that command
	f := strings.Fields(c.Substituted)
	name, args := f[0], f[1:]

	c.StartTime = time.Now()
	cmd := exec.CommandContext(jobCtx, name, args...)
	c.EndTime = time.Now()
	c.RunTime = c.EndTime.Sub(c.StartTime)
	c.RunTimePrintable = c.RunTime.String()

	cmd.Stdout = &outb
	cmd.Stderr = &errb

	c.Status = Running
	err := cmd.Run()

	if err != nil {
		c.Status = Errored
		if jobCtx.Err() == context.DeadlineExceeded {
			// TODO clean this up
			// return fmt.Errorf("command timed out: %w", err)
			c.Status = TimedOut
			slog.Info(fmt.Sprintf("command timed out: %v\n", err))
		}
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

// TODO don't pass in cmdList, just its length.
func getPBar(cmdListLen int, flags Flags) *progressbar.ProgressBar {
	pbar := progressbar.NewOptions(cmdListLen,
		progressbar.OptionSetVisibility(flags.Pbar),
		progressbar.OptionSetItsString("jobs"), // doesn't do anything, don't know why
		progressbar.OptionShowCount(),
		progressbar.OptionSetWriter(os.Stderr),
	)

	pbar.RenderBlank() // to get it to render at 0% before any job finishes

	return pbar

}

func commandLoop(loopCtx context.Context, loopCancel context.CancelFunc, commandsToRun CommandList, flags Flags) (CommandList, time.Duration) {

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
	pbar := getPBar(len(commandsToRun), flags)

	// launch all goroutines

	for _, c := range commandsToRun {

		go func() {
			tokens <- struct{}{} // get permission to start

			// create jobCtx and pass it in
			// workerCtx, workerCancel := context.WithTimeout(mainCtx, 5*time.Second)

			// TODO: what if flags.JobTimeout is zero?  need magic here?
			//   duration is int64 so just set it to that? 290 years.
			jobCtx, jobCancel := context.WithTimeout(loopCtx, flags.JobTimeout)
			c.JobTimeout = flags.JobTimeout

			executeSingleCommand(jobCtx, jobCancel, c)

			done <- c // report status.
			<-tokens  // return token when done.
		}()
	}

	// collect all goroutines

	doneList := CommandList{}
Outer:
	for completionCount != len(commandsToRun) {
		select {
		case c := <-done:
			completionCount += 1
			doneList = append(doneList, c)
			pbar.Add(1)
			if flags.FirstZero || (flags.Any && c.ReturnCode == 0) {
				// slog.Debug(fmt.Sprintf("returning %s", c.Arg))
				// this only returns the single command we're interested in regardless of what other commands have done.
				//  TODO is this what I want?  or do I want to return all commands but the other ones as NotStarted / whatever?
				// TODO: do I need to cancel all child contexts?  cancel the parent?  probably parent.
				break Outer
			}

		case <-loopCtx.Done():
			//fmt.Fprintf(os.Stderr, "global timeout popped, %v jobs done", len(completedCommands))
			slog.Info(fmt.Sprintf("global timeout popped, %v jobs done", len(completedCommands)))
			break Outer
		}
	}
	// Outer: breaks here

	loopCancel() // is this it?

	// sort doneList by completion time so .commands[0] is the fastest.
	sort.Slice(doneList, func(i int, j int) bool {
		return doneList[i].RunTime < doneList[j].RunTime
	})

	pbar.Finish()          // don't know if I need this.
	time.Sleep(pbarFinish) // to let the pbar finish displaying.

	return doneList, pbarFinish
}

func setTimeouts(globalTimeoutString, jobTimeoutString string) (time.Duration, time.Duration, error) {
	var globalDuration, jobDuration time.Duration
	var err error

	globalDuration, err = time.ParseDuration(globalTimeoutString)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid global timeout %v %v", jobTimeoutString, err)
	}

	jobDuration, err = time.ParseDuration(jobTimeoutString)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid job timeout %v %v", jobTimeoutString, err)
	}

	// now the logic starts

	// if job is set and global is not
	//   then set global to infinite
	if jobDuration > 0 && globalDuration == 0 {
		globalDuration = math.MaxInt64
		return globalDuration, jobDuration, nil
	}

	// if global is set and job is not
	//  then set job to global
	if globalDuration > 0 && jobDuration == 0 {
		jobDuration = globalDuration
		return globalDuration, jobDuration, nil

	}

	// if they're both zero
	if globalDuration == 0 && jobDuration == 0 {
		globalDuration = math.MaxInt64
		jobDuration = math.MaxInt64
		return globalDuration, jobDuration, nil
	}

	// if they're both set
	// 		jobDuration must be <= globalDuration
	if jobDuration > 0 && globalDuration > 0 {
		if !(jobDuration <= globalDuration) {
			return 0, 0, fmt.Errorf("job timeout must be less than global timeout, %v %v", jobDuration, globalDuration)
		}
	}

	// is that it?

	return globalDuration, jobDuration, nil

}

func PopulateFlags(cmd *cobra.Command) Flags {

	var err error

	flags := Flags{}
	// I sure wish there was a cleaner way to do this

	flags.Token, _ = cmd.Flags().GetString("token")
	flags.FlagErrors, _ = cmd.Flags().GetBool("flag-errors")
	flags.FirstZero, _ = cmd.Flags().GetBool("first")
	flags.Pbar, _ = cmd.Flags().GetBool("pbar")
	flags.LogLevel, _ = cmd.Flags().GetString("log")

	flags.Any, _ = cmd.Flags().GetBool("any")

	// TODO: break this into a separate function
	flags.ConcurrentJobLimit, _ = cmd.Flags().GetString("concurrent")

	switch flags.ConcurrentJobLimit {
	case "cpu", "1x":
		flags.GoroutineLimit = runtime.NumCPU()
	case "2x":
		flags.GoroutineLimit = runtime.NumCPU() * 2
	default:
		x, err := strconv.Atoi(flags.ConcurrentJobLimit)
		if err != nil || x == 0 {
			slog.Error(fmt.Sprintf("Invalid concurrency level: %s\n", flags.ConcurrentJobLimit))
			os.Exit(1)
		}
		flags.GoroutineLimit = x
	}

	globalTimeoutString, _ := cmd.Flags().GetString("timeout")
	jobTimeoutString, _ := cmd.Flags().GetString("job-timeout")
	flags.Timeout, flags.JobTimeout, err = setTimeouts(globalTimeoutString, jobTimeoutString)

	if err != nil {
		slog.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}

	return flags
}

func buildListOfCommands(command string, targets []string, token string) (CommandList, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.

	var ret CommandList
	var id JobID

	for _, target := range targets {
		slog.Debug(fmt.Sprintf("buildListOfCommands: target %q", target))
		x := Command{}
		x.Arg = target
		x.Substituted = strings.ReplaceAll(command, token, target)
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

	slog.Debug(fmt.Sprintf("buildListOfCommands: returning %q %v", ret, nil))
	return ret, nil
}
