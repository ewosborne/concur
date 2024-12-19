package infra

import (
	"fmt"
	"strings"
	"sync"
)

type Command struct {
	Original    string
	Substituted string
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

var wg sync.WaitGroup

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
		do_all(cmdList)
	} else if flags.Any {
		do_any(cmdList)
	}
}

// done

// 	for _, x := range cmdList.Commands {
// 		wg.Add(1)
// 		go func(Command) {
// 			defer wg.Done()
// 			fmt.Printf("%+v\n", x)
// 		}(x)
// 	}

// 	wg.Wait()
// }

func do_all(cmdList CommandList) {
	fmt.Println("in do_all with", cmdList)
}

func do_any(cmdList CommandList) {
	fmt.Println("in do_any with", cmdList)
}

func buildListOfCommands(command string, hosts []string) (CommandList, error) {
	// TODO I don't need a full template engine but should probably have something cooler than this.

	var ret CommandList
	for _, host := range hosts {
		x := Command{}
		x.Original = command
		x.Substituted = strings.ReplaceAll(command, "{{ host }}", host)
		ret.Commands = append(ret.Commands, x)
	}
	return ret, nil
}
