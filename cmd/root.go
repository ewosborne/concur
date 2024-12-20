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
	Long: `Concur is a replacement for the parts I like of GNU Parallel. Sensible defaults which assume blocking on network or I/O, straightforward syntax. No weird 
	\'will cite\' silliness, no $70 manual or 20-minute videos to watch, no CPAN imports to support CSV, and no Perl management. 
	Is parallel better at all the corner cases? Sure. But not for my use case. `,
	RunE: ConcurCmdE,
}

func ConcurCmdE(cmd *cobra.Command, args []string) error {

	command := args[0]
	opts := args[1:]
	flags := infra.Flags{}

	// I sure wish there was a cleaner way to do this
	flags.Any, _ = cmd.Flags().GetBool("any")
	flags.All, _ = cmd.Flags().GetBool("all")
	flags.ConcurrentLimit, _ = cmd.Flags().GetInt("concurrent")
	flags.Timeout, _ = cmd.Flags().GetInt64("timeout")

	// flags := make(map[string]string)
	// cmd.Flags().VisitAll(func(flag *pflag.Flag) {
	// 	flags[flag.Name] = flag.Value.String()
	// })

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
	rootCmd.Flags().Bool("all", false, "All commands")
	rootCmd.MarkFlagsMutuallyExclusive("any", "all")
	rootCmd.MarkFlagsOneRequired("any", "all") // TODO this isn't quite what I want.

	rootCmd.Flags().IntP("concurrent", "c", 128, "Number of concurrent processes (0 = no limit)")
	rootCmd.Flags().Int64P("timeout", "t", 90, "Timeout in seconds (0 for no timeout)")
}
