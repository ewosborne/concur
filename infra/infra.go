package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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

// wtf this results in unused write but setting it directly doesn't.  why?
// func (c Command) StartTimestamp() {
// 	c.StartTime = time.Now()
// }

func Do(command string, hosts []string, flags Flags) {
	// do all the heavy lifting here
	systemStartTime := time.Now()

	// TODO: pass flags in as float in seconds, convert to integer msec
	fmt.Println("flags is", flags.Timeout)
	t := time.Duration(flags.Timeout) * time.Millisecond
	fmt.Println("timeout", t)
	ctx, cancelCtx := context.WithTimeout(context.Background(), t)
	defer cancelCtx()

	fmt.Println("args", hosts, len(hosts))
	fmt.Printf("flags %+v\n", flags)

	// build a list of commands
	cmdList, err := buildListOfCommands(command, hosts)
	if err != nil {
		panic(err) // TODO fix
	}

	// go run the things
	completedCommands := start_command_loop(ctx, cmdList, flags)

	// presentation and cleanup
	systemEndTime := time.Now()
	systemRunTime := systemEndTime.Sub(systemStartTime)

	//fmt.Println("all done")
	for _, c := range completedCommands {
		//fmt.Println(c.Arg, c.RunTime)
		fmt.Println(c)
	}

	fmt.Println("OVERAL RUNTIME", systemRunTime)

}

func execute(ctx context.Context, c *Command) error {

	// TODO deal with breaking this into the command to run and its arguments
	f := strings.Fields(c.Substituted)
	name, args := f[0], f[1:]

	fmt.Println("executing", name, args, len(args))

	cmd := exec.CommandContext(ctx, name, args...)

	// TODO test stderr, make sure this works ok.

	var outb, errb strings.Builder
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	if err != nil {
		fmt.Println("ERROR HERE", err)
		return err
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

			fmt.Println("running command", c.Arg)

			// test: sleep for 0.1-2.6 sec
			err := execute(ctx, c)
			if err != nil {
				// TODO don't panic
				panic(fmt.Sprintf("error running command: %v %v", c.Arg, err))
			}
			c.EndTime = time.Now()
			rt := c.EndTime.Sub(c.StartTime)
			a, err := time.ParseDuration(rt.String())
			if err != nil {
				panic(err) // TODO
			}
			c.RunTime = a.String()

			//fmt.Println("in gofunc, command is done", c.Host, "runtime", c.RunTime)

			// wtf StartTime ends up print ok but not EndTime

			//fmt.Println(c)
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
				fmt.Println("ALL", len(cmdList), "COMMANDS DONE")
				//os.Exit(0) // TODO better
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
		x.Substituted = strings.ReplaceAll(command, "{{ arg }}", host)

		ret = append(ret, &x)
	}

	// mix them up just so there's no ordering depedency if they all take about the same time. otherwise the first one in the list
	//   tends to be the one we return first with --any.
	rand.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})

	return ret, nil
}
