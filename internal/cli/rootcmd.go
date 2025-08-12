package mailchatcli

import "github.com/spf13/cobra"

// GetRootCmd returns the root command for client functionality
func GetRootCmd() *cobra.Command {
	return rootCmd
}