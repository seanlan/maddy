package cmd

import (
	"github.com/spf13/cobra"

	mailchatdcmd "github.com/dsoftgames/MailChat/cmd/MailChatd/cmd"
	mailchatcli "github.com/dsoftgames/MailChat/internal/cli"
)

// NewRootCmd creates a new root command for MailChat
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "MailChat",
		Short: "MailChat unified command line interface",
		Long:  "MailChat provides both server daemon and client functionality in a single binary",
	}

	// Get server and client commands
	serverCmd := mailchatdcmd.NewRootCmd()
	clientCmd := mailchatcli.GetRootCmd()
	
	// Add all server subcommands directly to root
	for _, cmd := range serverCmd.Commands() {
		// Handle version command conflict by renaming server version to node-version
		if cmd.Use == "version" {
			cmd.Use = "node-version"
			cmd.Short = "Print the blockchain node version information"
		}
		
		// Ensure the command inherits the parent's PersistentPreRunE
		if cmd.PersistentPreRunE == nil && serverCmd.PersistentPreRunE != nil {
			cmd.PersistentPreRunE = serverCmd.PersistentPreRunE
		}
		
		rootCmd.AddCommand(cmd)
	}
	
	// Add all client subcommands directly to root  
	for _, cmd := range clientCmd.Commands() {
		// Handle version command conflict by renaming client version to mail-version
		if cmd.Use == "version" {
			cmd.Use = "mail-version"
			cmd.Short = "Print the mail server version information"
		}
		rootCmd.AddCommand(cmd)
	}

	return rootCmd
}