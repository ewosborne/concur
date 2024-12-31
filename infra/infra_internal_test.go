package infra

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TODO
func Test_executeSingleCommand(t *testing.T) {
	// func executeSingleCommand(jobCtx context.Context, jobCancel context.CancelFunc, c *Command)
}

func Test_getPBar(t *testing.T) {
	// func getPBar(cmdListLen int, flags Flags) *progressbar.ProgressBar
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

}

// TODO more test cases because setTimeouts has some ugly logic in it and I expect this to fail sometimes
func Test_setTimeouts(t *testing.T) {
	// func setTimeouts(globalTimeoutString, jobTimeoutString string) (time.Duration, time.Duration, error)

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
		{ // first test case
			global:     "7s",
			job:        "42s",
			expectPass: false,
		}, // end first test case
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
		}, // test case
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
