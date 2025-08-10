package mailcoincli

import (
	"fmt"
	"os"

	"mailcoin/framework/log"

	"github.com/spf13/cobra"
)

var rootCmd *cobra.Command

func init() {
	rootCmd = &cobra.Command{
		Use:   "mailcoin",
		Short: "composable all-in-one mail server",
		Long: `Maddy is Mail Transfer agent (MTA), Mail Delivery Agent (MDA), Mail Submission
Agent (MSA), IMAP server and a set of other essential protocols/schemes
necessary to run secure email server implemented in one executable.

This executable can be used to start the server ('run') and to manipulate
databases used by it (all other subcommands).`,
	}

	// Add hidden utility commands
	generateManCmd := &cobra.Command{
		Use:    "generate-man",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("generate-man not implemented yet for cobra")
		},
	}

	generateFishCompletionCmd := &cobra.Command{
		Use:    "generate-fish-completion",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenFishCompletion(os.Stdout, true)
		},
	}

	rootCmd.AddCommand(generateManCmd, generateFishCompletionCmd)
}

func AddGlobalStringFlag(name, usage, envVar, defaultValue string, dest *string) {
	if envVar != "" {
		rootCmd.PersistentFlags().StringVarP(dest, name, "", defaultValue, usage)
		if val := os.Getenv(envVar); val != "" {
			*dest = val
		}
	} else {
		rootCmd.PersistentFlags().StringVarP(dest, name, "", defaultValue, usage)
	}
}

func AddGlobalBoolFlag(name, usage string, dest *bool) {
	rootCmd.PersistentFlags().BoolVarP(dest, name, "", false, usage)
}

func AddSubcommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)

	if cmd.Name() == "run" {
		// Backward compatibility hack to start the server as just ./mailcoin
		rootCmd.RunE = cmd.RunE
	}
}

// Temporary compatibility function - remove after migration
func AddSubcommandLegacy(cmd interface{}) {
	// This is a no-op for now to allow compilation
	// Legacy commands need to be converted to cobra
}

// AddGlobalFlag provides compatibility with debug flags
func AddGlobalFlag(flag interface{}) {
	// This function handles legacy debug flags that are only compiled under special conditions
	// For now, these are ignored in the cobra migration
}

// RunWithoutExit is like Run but returns exit code instead of calling os.Exit
// To be used in mailcoin.cover.
func RunWithoutExit() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func Run() {

	if err := rootCmd.Execute(); err != nil {
		log.DefaultLogger.Error("rootCmd.Execute failed", err)
		os.Exit(1)
	}
}
