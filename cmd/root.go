/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/ewosborne/concur/infra"

	"github.com/spf13/cobra"
)

var logLevelFlag string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "concur <command string> <list of hosts> [flags]",
	Short: "Run commands concurrently",
	RunE:  ConcurCmdE,
}

func ConcurCmdE(cmd *cobra.Command, args []string) error {

	switch len(args) {
	case 0, 1: // 0 == command name only, 1 == string to run in but nothing to sub into it
		cmd.Help()
		os.Exit(1)
	}

	command := args[0]
	opts := args[1:]
	flags := infra.PopulateFlags(cmd)

	res := infra.Do(command, opts, flags)
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
	//
	rootCmd.Flags().Bool("any", false, "Return any (the first) command with exit code of zero")
	rootCmd.Flags().Bool("first", false, "First command regardless of exit code")

	rootCmd.Flags().StringP("concurrent", "c", "128", "Number of concurrent processes (0 = no limit), 'cpu' = one job per cpu core")
	rootCmd.Flags().Int64P("timeout", "t", 0, "Timeout in sec (0 default for no timeout)")
	rootCmd.Flags().StringP("token", "", "{{1}}", "Token to match for replacement")
	rootCmd.Flags().BoolP("flag-errors", "", false, "Print a message to stderr for all executed commands with an exit code other than zero")
	rootCmd.Flags().BoolP("pbar", "p", false, "Display a progress bar which ticks up once per completed job")
	rootCmd.Flags().BoolP("silent", "s", false, "Silent output")

	rootCmd.MarkFlagsMutuallyExclusive("pbar", "silent")

	rootCmd.PersistentFlags().StringVarP(&logLevelFlag, "log level", "l", "", "Enable debug mode (one of d, i, w, e)")
	var logLevel slog.Level
	var outStream io.Writer = os.Stderr

	// is there a better way to do this?  TODO
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {

		// TODO maybe make this a fixed set of options somehow?
		switch logLevelFlag {
		case "w":
			logLevel = slog.LevelWarn
		case "e":
			logLevel = slog.LevelError
		case "i":
			logLevel = slog.LevelInfo
		case "d":
			logLevel = slog.LevelDebug
		case "":
			outStream = io.Discard
		default:
			// can't log this because it's about setting logs..
			fmt.Fprintf(os.Stderr, "Invalid debug level: %s\n", logLevelFlag)
			os.Exit(1)
		}

		logger := slog.New(slog.NewTextHandler(outStream, &slog.HandlerOptions{
			AddSource: true,
			Level:     logLevel,
		}))
		slog.SetDefault(logger)
	}

}

func SetVersionInfo(version string) {
	rootCmd.Version = version
}
