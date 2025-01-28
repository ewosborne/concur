package infra

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// TODO
func Test_executeSingleCommand(t *testing.T) {
	// func executeSingleCommand(jobCtx context.Context, jobCancel context.CancelFunc, c *Command)

	t.Parallel()
	// needs from c: Substituted. and it fiddles with stuff on the way back in.
	ctx, ctxCancel := context.WithCancel(context.Background())

	c := Command{
		Substituted: "echo hello",
	}

	executeSingleCommand(ctx, ctxCancel, &c)

	// now what?  sanity check stuff

	if c.Status != Finished {
		t.Errorf("status should be Finished but is instead %q", c.Status)
	}

	if c.RunTime <= 0 {
		t.Errorf("runtime should be >0 but is instead %q", c.RunTime)
	}

	if c.ReturnCode != 0 {
		t.Errorf("return code from echo should be 0 but is instead %v", c.ReturnCode)
	}

	if c.Stdout[0] != "hello" || len(c.Stdout) != 2 { // len 2 because newline is a separate slice item
		t.Errorf("stdout is wonky: %q", c.Stdout)
	}

}

func Test_getPBar(t *testing.T) {
	t.Parallel()
	want := "*progressbar.ProgressBar"

	tmp := getPBar(42, Flags{})
	got := fmt.Sprintf("%T", tmp)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("getPBar type mismatch:\n%s", diff)
	}
}

// TODO
func Test_commandLoop(t *testing.T) {
	// func commandLoop(loopCtx context.Context, loopCancel context.CancelFunc, commandsToRun CommandList, flags Flags) (CommandList, time.Duration)

	t.Parallel()

	ctx, ctxCancel := context.WithCancel(context.Background())

	jd, _ := time.ParseDuration("5s")
	cmdList := CommandList{
		&Command{
			ID:          0,
			Status:      TBD,
			Substituted: "echo hello",
			Arg:         "hello",
		}, // command
		&Command{
			ID:          0,
			Status:      TBD,
			Substituted: "ping -c 1 www.mit.edu",
			Arg:         "www.mit.edu",
		},
	} //CommandList

	flags := Flags{
		Timeout:        3,
		JobTimeout:     jd,
		GoroutineLimit: len(cmdList),
	}

	resList, runtime := commandLoop(ctx, ctxCancel, cmdList, flags)
	// t.Log(resList, runtime)
	//  not sure what else to check in these two here
	if runtime < 0 {
		t.Errorf("runtime is too small %q", runtime)
	}

	for _, cmd := range resList {
		if cmd.Status != Finished {
			t.Errorf("expected status Finished but got %q", cmd.Status)
		}
	}

}

// TODO something is off - sometimes when I pass in zero timeout I get infinite back, that's by design, but how do I test it?
func Test_setTimeouts(t *testing.T) {
	// func setTimeouts(globalTimeoutString, jobTimeoutString string) (time.Duration, time.Duration, error)

	t.Parallel()

	testCases := []struct {
		global     string
		job        string
		expectPass bool
	}{
		{ // first test case
			global:     "8s",
			job:        "4s",
			expectPass: true,
		}, // end first test case
		{ // second test case
			global:     "7s",
			job:        "42s",
			expectPass: false,
		}, // end second test case
		{
			global:     "0s",
			job:        "2s",
			expectPass: true,
		},
		{
			global:     "10s",
			job:        "10s",
			expectPass: true,
		},
		{
			global:     "0s",
			job:        "0s",
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		// now call the thing
		global, job, err := setTimeouts(tc.global, tc.job)

		if tc.expectPass == true && err != nil {
			t.Errorf("error %q when there should be none with %q %q", err, global, job)
		}

		if tc.expectPass == false && err == nil {
			t.Errorf("no error seen when there should be one with %q %q", tc.global, tc.job)
		}

	}
}

func Test_buildListOfCommands(t *testing.T) {

	t.Parallel()
	testCases := []struct {
		command  string
		targets  []string
		token    string
		expected CommandList
	}{
		{ // First test case
			command: "echo {{1}}",
			targets: []string{"hello"},
			token:   "{{1}}",
			expected: CommandList{
				&Command{
					ID:          0,
					Status:      TBD,
					Substituted: "echo hello",
					Arg:         "hello",
				}, // command
			}, //command list
		}, // first test case
		{ // Second test case
			command: "ping __SUBS__",
			targets: []string{"www.mit.edu"},
			token:   "__SUBS__",
			expected: CommandList{
				&Command{
					ID:          0,
					Status:      TBD,
					Substituted: "ping www.mit.edu",
					Arg:         "www.mit.edu",
				},
			},
		},
	}

	for _, tc := range testCases {
		got, err := buildListOfCommands(tc.command, tc.targets, tc.token)
		if err != nil {
			t.Errorf("got some sort of error from buildListOfCommands %q", err)
		}

		if diff := cmp.Diff(tc.expected, got); diff != "" {
			t.Errorf("diff\n%s", diff)
		}

	}
}
