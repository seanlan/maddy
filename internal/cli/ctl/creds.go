/*
Maddy Mail Server - Composable all-in-one email server.
Copyright © 2019-2020 Max Mazurov <fox.cpp@disroot.org>, Maddy Mail Server contributors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package ctl

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"github.com/dsoftgames/MailChat/internal/auth/pass_table"
	maddycli "github.com/dsoftgames/MailChat/internal/cli"
)

func init() {
	credsCmd := &cobra.Command{
		Use:   "creds",
		Short: "User credentials management",
		Long: `These subcommands can be used to manage local user credentials for any
authentication module supported by MailChat.

The corresponding authentication module should be configured in mailchat.conf and be
defined in a top-level configuration block. By default, the name of that
block should be local_authdb but this can be changed using --cfg-block
flag for subcommands.`,
	}

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List user accounts",
		RunE:  credsList,
	}
	listCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	listCmd.Flags().Bool("quiet", false, "Do not print 'No users.' message")

	// Create subcommand  
	createCmd := &cobra.Command{
		Use:     "create USERNAME",
		Short:   "Create user account",
		Args:    cobra.ExactArgs(1),
		RunE:    credsCreate,
	}
	createCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	createCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin")
	createCmd.Flags().String("hash", "bcrypt", "Hash algorithm to use")
	createCmd.Flags().Int("bcrypt-cost", bcrypt.DefaultCost, "Bcrypt cost value")

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:     "remove USERNAME", 
		Short:   "Delete user account",
		Args:    cobra.ExactArgs(1),
		RunE:    credsRemove,
	}
	removeCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	removeCmd.Flags().BoolP("yes", "y", false, "Don't ask for confirmation")

	// Password subcommand
	passwordCmd := &cobra.Command{
		Use:     "password USERNAME",
		Short:   "Change user password", 
		Args:    cobra.ExactArgs(1),
		RunE:    credsPassword,
	}
	passwordCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	passwordCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin")

	credsCmd.AddCommand(listCmd, createCmd, removeCmd, passwordCmd)
	maddycli.AddSubcommand(credsCmd)
}

func credsList(cmd *cobra.Command, args []string) error {
	be, err := openUserDB(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	list, err := be.ListUsers()
	if err != nil {
		return err
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if len(list) == 0 && !quiet {
		fmt.Fprintln(os.Stderr, "No users.")
	}

	for _, user := range list {
		fmt.Println(user)
	}
	return nil
}

func credsCreate(cmd *cobra.Command, args []string) error {
	be, err := openUserDB(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]

	var pass string
	if cmd.Flags().Changed("password") {
		pass, _ = cmd.Flags().GetString("password")
	} else {
		var err error
		pass, err = maddycli.ReadPassword("Enter password for new user")
		if err != nil {
			return err
		}
	}

	if beHash, ok := be.(*pass_table.Auth); ok {
		hashAlgo, _ := cmd.Flags().GetString("hash")
		bcryptCost, _ := cmd.Flags().GetInt("bcrypt-cost")
		return beHash.CreateUserHash(username, pass, hashAlgo, pass_table.HashOpts{
			BcryptCost: bcryptCost,
		})
	} else if cmd.Flags().Changed("hash") || cmd.Flags().Changed("bcrypt-cost") {
		return fmt.Errorf("--hash cannot be used with non-pass_table credentials DB")
	} else {
		return be.CreateUser(username, pass)
	}
}

func credsRemove(cmd *cobra.Command, args []string) error {
	be, err := openUserDB(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		if !maddycli.Confirmation("Are you sure you want to delete this user account?", false) {
			return fmt.Errorf("cancelled")
		}
	}

	return be.DeleteUser(username)
}

func credsPassword(cmd *cobra.Command, args []string) error {
	be, err := openUserDB(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]

	var pass string
	if cmd.Flags().Changed("password") {
		pass, _ = cmd.Flags().GetString("password")
	} else {
		var err error
		pass, err = maddycli.ReadPassword("Enter new password")
		if err != nil {
			return err
		}
	}

	return be.SetUserPassword(username, pass)
}