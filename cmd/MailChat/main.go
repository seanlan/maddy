package main

import (
	"fmt"
	"os"

	_ "github.com/dsoftgames/MailChat"
	"github.com/dsoftgames/MailChat/cmd/MailChat/cmd"
	_ "github.com/dsoftgames/MailChat/internal/cli/ctl"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}