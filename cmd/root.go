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
	Use:   "concur",
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
	flags := populateFlags(cmd)

	infra.Do(command, opts, flags)
	return nil
}

func populateFlags(cmd *cobra.Command) infra.Flags {
	flags := infra.Flags{}
	// I sure wish there was a cleaner way to do this
	flags.Any, _ = cmd.Flags().GetBool("any")
	flags.ConcurrentLimit, _ = cmd.Flags().GetInt("concurrent")
	flags.Timeout, _ = cmd.Flags().GetInt64("timeout")
	flags.Token, _ = cmd.Flags().GetString("token")
	flags.FlagErrors, _ = cmd.Flags().GetBool("flag-errors")
	return flags
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Bool("any", false, "Any (first) command")
	//	rootCmd.MarkFlagsMutuallyExclusive("any", "all")
	//	rootCmd.MarkFlagsOneRequired("any", "all") // TODO this isn't quite what I want.

	rootCmd.Flags().IntP("concurrent", "c", 128, "Number of concurrent processes (0 = no limit)")
	rootCmd.Flags().Int64P("timeout", "t", 90_000, "Timeout in msec (0 for no timeout)")
	rootCmd.Flags().StringP("token", "", "{{1}}", "Token to match for replacement")
	rootCmd.Flags().BoolP("flag-errors", "", false, "Print a message to stderr for all executed commands with an exit code other than zero")
	rootCmd.PersistentFlags().StringVarP(&logLevelFlag, "log level", "l", "", "Enable debug mode (one of d, i, w, e)")

	// debugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime)

	// // need PreRun because flags aren't parsed until a command is run.
	var logLevel slog.Level
	var outStream io.Writer = os.Stderr
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
