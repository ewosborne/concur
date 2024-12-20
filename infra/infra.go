package infra

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type Command struct {
	Original    string
	Substituted string
	Host        string
	Stdout      string // placeholder
	Stdin       string // placeholder
	StartTime   time.Time
	EndTime     time.Time
	RunTime     time.Duration
}

type Flags struct {
	Any             bool
	All             bool
	ConcurrentLimit int
	Timeout         int64
}

// is this worth it?  Maybe for a String() method?
// doing away with it for now

// type CommandList struct {
// 	Commands []*Command
// }

func (c Command) String() string {
	return fmt.Sprintf(`
	 Original:%v
	 Substituted:%v
	 Stdout:%v
	 Stdin:%v
	 StartTime: %v
	 EndTime: %v
	 RunTime: %v
	 `, c.Original, c.Substituted, c.Stdout, c.Stdin, c.StartTime, c.EndTime, c.RunTime)
}

func Do(command string, hosts []string, flags Flags) {
	// do all the heavy lifting here
	startTime := time.Now()

	// TODO: pass flags in as float in seconds, convert to integer msec
	fmt.Println("flags is", flags.Timeout)
	t := time.Duration(flags.Timeout) * time.Millisecond
	fmt.Println("timeout", t)
	ctx, cancelCtx := context.WithTimeout(context.Background(), t)
	defer cancelCtx()

	fmt.Println("hosts", hosts, len(hosts))
	fmt.Printf("flags %+v\n", flags)

	// sanitize any vs. all until I can figure flag groups TODO
	if !flags.All && !flags.Any {
		flags.All = true
	}

	// build a list of commands
	// TODO maybe cmdList is a list of pointers to commands?
	cmdList, err := buildListOfCommands(command, hosts)
	if err != nil {
		panic(err) // TODO fix
	}

	// go run the things
	// TODO make sure I understand what needs to be pointers and where and why

	completedCommands := start_command_loop(ctx, cmdList, flags)

	fmt.Println("all done")
	endTime := time.Now()
	runTime := endTime.Sub(startTime)
	for _, c := range completedCommands {
		fmt.Println(c.Host, c.RunTime)
	}

	fmt.Println("OVERAL START/END/RUN", startTime, endTime, runTime)

}

func start_command_loop(ctx context.Context, cmdList []*Command, flags Flags) []*Command {
	//fmt.Println("in start_command_loop with", cmdList)

	var tokens = make(chan struct{}, flags.ConcurrentLimit)
	var done = make(chan *Command)   // where a command goes when it's done
	var completedCommands []*Command // count all the done processes

	// launch each command
	for _, c := range cmdList {

		go func() {
			tokens <- struct{}{} // get permission to start
			c.StartTime = time.Now()

			fmt.Println("running command", c.Host)
			time.Sleep(time.Duration(rand.Intn(1500)) * time.Millisecond)
			time.Sleep(time.Duration(1 * time.Second))
			c.EndTime = time.Now()
			c.RunTime = c.EndTime.Sub(c.StartTime)
			//fmt.Println(" after", c)
			fmt.Println("in gofunc, command is done", c.Host, "runtime", c.RunTime)
			done <- c // report status
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

func buildListOfCommands(command string, hosts []string) ([]*Command, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.

	var ret []*Command
	for _, host := range hosts {
		x := Command{}
		x.Original = command
		x.Host = host
		x.Substituted = strings.ReplaceAll(command, "{{ arg }}", host)

		ret = append(ret, &x)
	}
	return ret, nil
}
