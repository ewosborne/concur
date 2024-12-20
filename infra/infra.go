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
type CommandList struct {
	Commands []Command
}

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
	cmdList, err := buildListOfCommands(command, hosts)
	if err != nil {
		panic(err) // TODO fix
	}

	// go run the things

	start_command_loop(ctx, cmdList, flags)

	fmt.Println("all done")
}

func start_command_loop(ctx context.Context, cmdList CommandList, flags Flags) {
	fmt.Println("in start_command_loop with", cmdList)

	var tokens = make(chan struct{}, flags.ConcurrentLimit)
	var done = make(chan *Command) // where a command goes when it's done
	var doneCounter int            // count all the done processes

	// launch each command
	for _, c := range cmdList.Commands {

		go func() {
			tokens <- struct{}{} // get permission to start
			fmt.Println("running command", c.Host)

			c.StartTime = time.Now()

			fmt.Println("running command, before", c)
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			time.Sleep(time.Duration(1 * time.Second))
			c.EndTime = time.Now()
			c.RunTime = c.EndTime.Sub(c.StartTime)
			fmt.Println(" after", c)

			fmt.Println(c.Host, "TIMES", c.StartTime, c.EndTime, c.RunTime)

			fmt.Println("in gofunc, command is done", c.Host, "runtime", c.RunTime)
			done <- &c // report status
			<-tokens   // return token when done.
		}()

	}

	// TODO should break this loop out into another function I guess.
	for {
		select {
		case c := <-done:
			doneCounter += 1

			fmt.Println(c.Host, "command is done")

			if flags.Any {
				fmt.Println("first command returned, exiting")
				//os.Exit(0) // TODO just return or something
				return
			} // otherwise flags.All so don't exit loop

			if doneCounter == len(cmdList.Commands) {
				fmt.Println("ALL", len(cmdList.Commands), "COMMANDS DONE")
				//os.Exit(0) // TODO better
				return
			}
		case <-ctx.Done():
			msg := fmt.Sprintf("context popped, %v jobs done", doneCounter)
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
		x.Host = host
		x.Substituted = strings.ReplaceAll(command, "{{ arg }}", host)

		ret.Commands = append(ret.Commands, x)
	}
	return ret, nil
}
