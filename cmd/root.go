/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ewosborne/concur/infra"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "concur <command string> <list of hosts> [flags]",
	Short: "Run commands concurrently",
	RunE:  ConcurCmdE,
}

// return whether there's something to read on stdin
func getArgsFromStdin() ([]string, bool) {

	fi, _ := os.Stdin.Stat()

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		//fmt.Println("reading from stdin")

		bytes, _ := io.ReadAll(os.Stdin)
		str := string(bytes)

		// .Fields() breaks the input string into separate words.  That seems ok but maybe it's not quite right?
		return strings.Fields(str), true
	}

	return nil, false
}

// TODO this function does too much and needs to be broken out into testable bits.
func ConcurCmdE(cmd *cobra.Command, args []string) error {

	var targets []string
	var template string

	stdinArgs, ok := getArgsFromStdin()

	if ok {
		targets = stdinArgs
		template = args[0]
	} else {
		switch len(args) {
		case 0, 1: // 0 == command name only, 1 == string to run in but nothing to sub into it
			cmd.Help()
			os.Exit(1)
		}
		template = args[0]
		targets = args[1:]
	}

	flags := infra.PopulateFlags(cmd)

	/* logs are called like

	slog.Info("hello slog info")
	slog.Debug("hello slog debug")
	slog.Error("hello slog error")

	*/

	// magic to set log level, this is about as clean as I can get it
	switch flags.LogLevel {
	case "d":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "i":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "w":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "e": // default
		slog.SetLogLoggerLevel(slog.LevelError)
	case "q": // quiet.  not sure I want this, what about errors which cause the program to exit?
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard,
			&slog.HandlerOptions{})))
	default:
		fmt.Fprintf(os.Stderr, "Invalid log level: %s\n", flags.LogLevel)
		os.Exit(1)
	}

	res := infra.Do(template, targets, flags)
	infra.ReportDone(res, flags)
	return nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	// NOTE: infra.PopulateFlags() and infra.Flag also need to be updated when flags are tweaked.
	//  I don't like that approach and should clean it up.

	rootCmd.Flags().Bool("any", false, "Return any (the first) job with exit code of zero")
	rootCmd.Flags().Bool("first", false, "First commanjobd regardless of exit code")

	rootCmd.Flags().StringP("concurrent", "c", "128",
		"Number of concurrent jobs (0 = no limit), 'cpu' or '1x' = one job per cpu core, '2x' = two jobs per cpu core")
	rootCmd.Flags().StringP("timeout", "t", "0", "Global timeout in time.Duration format (0 default for no timeout)")
	rootCmd.Flags().StringP("token", "", "{{1}}", "Token to match for replacement")
	rootCmd.Flags().BoolP("flag-errors", "", false, "Print a message to stderr for all completed jobs with an exit code other than zero")
	rootCmd.Flags().BoolP("pbar", "p", false, "Display a progress bar which ticks up once per completed job")
	rootCmd.Flags().StringP("job-timeout", "j", "0", "Per-job timeout in time.Duration format (0 default, must be <= global timeout)")
	rootCmd.Flags().StringP("log", "l", "e", "Enable debug mode (one of d, i, w, e, or q for quiet).")

}

func SetVersionInfo(version string) {
	rootCmd.Version = version
}

func TestscriptEntryPoint() int {
	Execute()
	return 0
}
