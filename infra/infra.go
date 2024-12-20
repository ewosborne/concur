package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Command struct {
	Original    string    `json:"original"`
	Substituted string    `json:"substituted"`
	Arg         string    `json:"arg"`
	Stdout      string    `json:"stdout"`
	Stdin       string    `json:"stdin"`
	Stderr      string    `json:"stderr"`
	StartTime   time.Time `json:"starttime"`
	EndTime     time.Time `json:"endtime"`
	RunTime     string    `json:"runtime"`
	ReturnCode  int       `json:"returncode"`
}

type Flags struct {
	Any             bool
	All             bool
	ConcurrentLimit int
	Timeout         int64
}

type CommandList []*Command

func (c Command) String() string {
	b, _ := json.MarshalIndent(c, "", " ") // TODO clean this up, particularly RunTime
	return string(b)

}

func Do(command string, substituteArgs []string, flags Flags) {
	// do all the heavy lifting here
	if len(substituteArgs) < 1 {
		fmt.Fprintf(os.Stderr, "ERROR: no arguments provided")
		os.Exit(1)
	}
	systemStartTime := time.Now()

	// TODO: pass flags in as float in seconds, convert to integer msec
	//fmt.Println("flags is", flags.Timeout)
	t := time.Duration(flags.Timeout) * time.Millisecond
	//fmt.Println("timeout", t)
	ctx, cancelCtx := context.WithTimeout(context.Background(), t)
	defer cancelCtx()

	//fmt.Println("args", substituteArgs, len(substituteArgs))
	//fmt.Printf("flags %+v\n", flags)

	// build a list of commands
	cmdList, err := buildListOfCommands(command, substituteArgs)
	if err != nil {
		panic(err) // TODO fix
	}

	// go run the things
	completedCommands := start_command_loop(ctx, cmdList, flags)

	// presentation and cleanup
	systemEndTime := time.Now()
	systemRunTime := systemEndTime.Sub(systemStartTime)

	reportDone(completedCommands, systemRunTime)
}

type Results struct {
	Commands CommandList
	Info     map[string]string
}

func reportDone(completedCommands CommandList, systemRunTime time.Duration) {

	// json
	//  top level: commands, info
	// commands contains output from each command
	// info contains stuff like runtime

	var res = Results{}
	res.Info = make(map[string]string)

	res.Commands = completedCommands
	res.Info["systemRunTime"] = systemRunTime.String()

	//fmt.Println("all done")
	// for _, c := range res["commands"] {
	// 	//fmt.Println(c.Arg, c.RunTime)
	// 	fmt.Println(c)
	// }

	results, err := json.MarshalIndent(res, "", " ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling results")
	}
	fmt.Println(string(results))

	//fmt.Println("OVERAL RUNTIME", res.Info["systemRunTime"])

}

func execute(ctx context.Context, c *Command) error {

	// TODO deal with breaking this into the command to run and its arguments
	f := strings.Fields(c.Substituted)
	name, args := f[0], f[1:]

	//fmt.Println("executing", name, args, len(args))

	cmd := exec.CommandContext(ctx, name, args...)

	// TODO test stderr, make sure this works ok.

	var outb, errb strings.Builder
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	c.ReturnCode = 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			c.ReturnCode = exitError.ExitCode()
			return err
		}
	}

	c.Stdout = outb.String()
	c.Stderr = errb.String()
	return nil
}

func start_command_loop(ctx context.Context, cmdList CommandList, flags Flags) CommandList {

	var tokens = make(chan struct{}, flags.ConcurrentLimit) // permission to run
	var done = make(chan *Command)                          // where a command goes when it's done
	var completedCommands CommandList                       // count all the done processes

	// launch each command
	for _, c := range cmdList {

		go func() {
			tokens <- struct{}{} // ge!jt permission to start
			c.StartTime = time.Now()

			//fmt.Println("running command", c.Arg)

			// test: sleep for 0.1-2.6 sec
			err := execute(ctx, c)
			if err != nil {
				//fmt.Fprintf(os.Stderr, "error running command: %v %v\n", c.Arg, err)
			}
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
	for {
		select {
		case c := <-done:
			completedCommands = append(completedCommands, c)

			//fmt.Println(c.Host, "command is done")

			if flags.Any {
				fmt.Println("first command returned, exiting")
				//os.Exit(0) // TODO just return or something
				return completedCommands
			} // otherwise flags.All so don't exit loop

			if len(completedCommands) == len(cmdList) {
				//fmt.Println("ALL", len(cmdList), "COMMANDS DONE")
				return completedCommands
			}
		case <-ctx.Done():
			msg := fmt.Sprintf("context popped, %v jobs done", len(completedCommands))
			panic(msg)
		}
	}
}

func buildListOfCommands(command string, hosts []string) (CommandList, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.

	var ret CommandList
	for _, host := range hosts {
		x := Command{}
		x.Original = command
		x.Arg = host
		x.Substituted = strings.ReplaceAll(command, "{{1}}", host)

		ret = append(ret, &x)
	}

	// mix them up just so there's no ordering depedency if they all take about the same time. otherwise the first one in the list
	//   tends to be the one we return first with --any.
	rand.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})

	return ret, nil
}
