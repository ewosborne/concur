package infra

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Command struct {
	Original    string
	Substituted string
	Host        string
	Stdout      string // placeholder
	Stdin       string // placeholder
}

type Flags struct {
	Any        bool
	All        bool
	Concurrent int
}

// is this worth it?  Maybe for a String() method?
type CommandList struct {
	Commands []Command
}

func (c Command) String() string {
	return fmt.Sprintf(`
	 Original:%s
	 Substituted:%s
	 Stdout:%s
	 Stdin:%s
	 `, c.Original, c.Substituted, c.Stdout, c.Stdin)
}

func Do(command string, hosts []string, flags Flags) {
	// do all the heavy lifting here

	// fmt.Println("in infra")
	// fmt.Println("command", command)
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

	if flags.All {
		do_all(cmdList, flags)
	} else if flags.Any {
		do_any(cmdList, flags)
	}
}

func do_all(cmdList CommandList, flags Flags) {
	//fmt.Println("in do_all with", cmdList)

	var wg sync.WaitGroup
	var tokens = make(chan struct{}, flags.Concurrent)
	for _, x := range cmdList.Commands {
		wg.Add(1)

		go func(Command) {
			defer wg.Done()
			tokens <- struct{}{}
			run_command(x)
			//fmt.Printf("%+v\n", x)
			<-tokens
		}(x)
	}
	wg.Wait()
}

func run_command(c Command) {
	// TODO
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	fmt.Println("running command", c)
}

func do_any(cmdList CommandList, flags Flags) {
	// TODO
	//fmt.Println("in do_any with", cmdList)

	var wg sync.WaitGroup
	var tokens = make(chan struct{}, flags.Concurrent)
	var single = make(chan Command)

	// launch everything
	for _, x := range cmdList.Commands {
		wg.Add(1)

		go func(Command) {
			defer wg.Done()
			tokens <- struct{}{}
			run_command(x)
			<-tokens
			single <- x
		}(x)
	}

	// wait for the first one to finish
	cc := <-single
	fmt.Println("execution host", cc.Host)

}

func buildListOfCommands(command string, hosts []string) (CommandList, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.

	var ret CommandList
	for _, host := range hosts {
		x := Command{}
		x.Original = command
		x.Host = host
		x.Substituted = strings.ReplaceAll(command, "{{ host }}", host)

		ret.Commands = append(ret.Commands, x)
	}
	return ret, nil
}
