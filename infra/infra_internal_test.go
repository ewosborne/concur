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
func Test_setTimeouts(t *testing.T) {
	// func setTimeouts(cmd *cobra.Command) (time.Duration, time.Duration, error)

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
				},
			},
		},
		{ // Second
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
