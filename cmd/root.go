/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/ewosborne/concur/infra"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "concur",
	Short: "Run commands concurrently",
	RunE:  ConcurCmdE,
}

func ConcurCmdE(cmd *cobra.Command, args []string) error {

	command := args[0]
	opts := args[1:]
	flags := infra.Flags{}

	// I sure wish there was a cleaner way to do this
	flags.Any, _ = cmd.Flags().GetBool("any")
	flags.ConcurrentLimit, _ = cmd.Flags().GetInt("concurrent")
	flags.Timeout, _ = cmd.Flags().GetInt64("timeout")

	// does this make it easier?
	//flags.All = !flags.Any

	infra.Do(command, opts, flags)
	return nil
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
}
