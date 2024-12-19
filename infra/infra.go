package infra

import (
	"fmt"
	"strings"
)

type Command struct {
	Original    string
	Substituted string
	Stdout      string // placeholder
	Stdin       string // placeholder
}

type Flags struct {
	Any bool
	All bool
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
	// fmt.Println("in infra")
	// fmt.Println("command", command)
	fmt.Println("hosts", hosts, len(hosts))
	fmt.Printf("flags %+v\n", flags)

	// sanitize any vs. all
	//  sure would be nice if these were in a struct so I didn't have fake bools.
	// TODO
	if !flags.All && !flags.Any {
		flags.All = true
	}
	// do all the heavy lifting here

	// first build a list of commands
	cmdList, err := buildListOfCommands(command, hosts)
	if err != nil {
		panic(err) // TODO fix
	}

	for _, x := range cmdList.Commands {
		fmt.Printf("%+v\n", x)
	}
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
