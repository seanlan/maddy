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

	mailchatcli "github.com/dsoftgames/MailChat/internal/cli"
	"github.com/emersion/go-imap"
	"github.com/spf13/cobra"
)

func init() {
	imapMboxesCmd := &cobra.Command{
		Use:   "imap-mboxes",
		Short: "IMAP mailboxes (folders) management",
	}

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list USERNAME",
		Short: "Show mailboxes of user",
		Args:  cobra.ExactArgs(1),
		RunE:  mboxesList,
	}
	listCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	listCmd.Flags().BoolP("subscribed", "s", false, "List only subscribed mailboxes")

	// Create subcommand
	createCmd := &cobra.Command{
		Use:   "create USERNAME NAME",
		Short: "Create mailbox",
		Args:  cobra.ExactArgs(2),
		RunE:  mboxesCreate,
	}
	createCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	createCmd.Flags().String("special", "", "Set SPECIAL-USE attribute on mailbox; valid values: archive, drafts, junk, sent, trash")

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove USERNAME MAILBOX",
		Short: "Remove mailbox",
		Long:  "WARNING: All contents of mailbox will be irrecoverably lost.",
		Args:  cobra.ExactArgs(2),
		RunE:  mboxesRemove,
	}
	removeCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	removeCmd.Flags().BoolP("yes", "y", false, "Don't ask for confirmation")

	// Rename subcommand
	renameCmd := &cobra.Command{
		Use:   "rename USERNAME OLDNAME NEWNAME",
		Short: "Rename mailbox",
		Long:  "Rename may cause unexpected failures on client-side so be careful.",
		Args:  cobra.ExactArgs(3),
		RunE:  mboxesRename,
	}
	renameCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")

	imapMboxesCmd.AddCommand(listCmd, createCmd, removeCmd, renameCmd)
	mailchatcli.AddSubcommand(imapMboxesCmd)
}

func mboxesList(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	subscribed, _ := cmd.Flags().GetBool("subscribed")
	mboxes, err := u.ListMailboxes(subscribed)
	if err != nil {
		return err
	}

	for _, mbox := range mboxes {
		attrs := mbox.Attributes
		if len(attrs) == 0 {
			fmt.Printf("%s\n", mbox.Name)
		} else {
			fmt.Printf("%s", mbox.Name)
			for _, attr := range attrs {
				fmt.Printf(" %s", attr)
			}
			fmt.Println()
		}
	}

	return nil
}

func mboxesCreate(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	name := args[1]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	if cmd.Flags().Changed("special") {
		special, _ := cmd.Flags().GetString("special")
		var attr string
		switch special {
		case "archive":
			attr = imap.ArchiveAttr
		case "drafts":
			attr = imap.DraftsAttr
		case "junk":
			attr = imap.JunkAttr
		case "sent":
			attr = imap.SentAttr
		case "trash":
			attr = imap.TrashAttr
		default:
			return fmt.Errorf("unknown special-use attribute: %s", special)
		}

		if suu, ok := u.(SpecialUseUser); ok {
			return suu.CreateMailboxSpecial(name, attr)
		} else {
			return fmt.Errorf("backend does not support SPECIAL-USE IMAP extension")
		}
	}

	return u.CreateMailbox(name)
}

func mboxesRemove(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	name := args[1]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		if !mailchatcli.Confirmation("Are you sure you want to delete that mailbox?", false) {
			return fmt.Errorf("cancelled")
		}
	}

	return u.DeleteMailbox(name)
}

func mboxesRename(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	oldName := args[1]
	newName := args[2]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	return u.RenameMailbox(oldName, newName)
}