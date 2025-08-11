package mailchatcli

import (
	"os"

	"github.com/dsoftgames/MailChat/framework/log"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var rootCmd *cobra.Command

func init() {
	rootCmd = &cobra.Command{
		Use:   "MailChat",
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
		Short:  "Generate man page",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doc.GenManTree(rootCmd, nil, ".")
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
	rootCmd.PersistentFlags().StringVarP(dest, name, "", defaultValue, usage)
	if envVar != "" {
		if val := os.Getenv(envVar); val != "" {
			*dest = val
		}
	}
}

func AddGlobalBoolFlag(name, usage string, dest *bool) {
	rootCmd.PersistentFlags().BoolVarP(dest, name, "", false, usage)
}

func AddSubcommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)

	if cmd.Name() == "run" {
		// Backward compatibility hack to start the server as just ./MailChat
		rootCmd.RunE = cmd.RunE
	}
}


// RunWithoutExit is like Run but returns exit code instead of calling os.Exit
// To be used in MailChat.cover.
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
