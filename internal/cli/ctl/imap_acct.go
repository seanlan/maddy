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

	"github.com/dsoftgames/MailChat/framework/module"
	mailchatcli "github.com/dsoftgames/MailChat/internal/cli"
	"github.com/emersion/go-imap"
	"github.com/spf13/cobra"
)

type SpecialUseUser interface {
	CreateMailboxSpecial(name, specialUseAttr string) error
}

func init() {
	imapAcctCmd := &cobra.Command{
		Use:   "imap-acct",
		Short: "IMAP storage accounts management",
		Long: `These subcommands can be used to list/create/delete IMAP storage
accounts for any storage backend supported by MailChat.

The corresponding storage backend should be configured in mailchat.conf and be
defined in a top-level configuration block. By default, the name of that
block should be local_mailboxes but this can be changed using --cfg-block
flag for subcommands.

Note that in default configuration it is not enough to create an IMAP storage
account to grant server access. Additionally, user credentials should
be created using 'creds' subcommand.`,
	}

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List storage accounts",
		RunE:  imapAcctList,
	}
	listCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")

	// Create subcommand
	createCmd := &cobra.Command{
		Use:   "create USERNAME",
		Short: "Create IMAP storage account",
		Long: `In addition to account creation, this command
creates a set of default folder (mailboxes) with special-use attribute set.`,
		Args: cobra.ExactArgs(1),
		RunE: imapAcctCreate,
	}
	createCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	createCmd.Flags().Bool("no-specialuse", false, "Do not create special-use folders")
	createCmd.Flags().String("sent-name", "Sent", "Name of special mailbox for sent messages, use empty string to not create any")
	createCmd.Flags().String("trash-name", "Trash", "Name of special mailbox for trash, use empty string to not create any")
	createCmd.Flags().String("junk-name", "Junk", "Name of special mailbox for 'junk' (spam), use empty string to not create any")
	createCmd.Flags().String("drafts-name", "Drafts", "Name of special mailbox for drafts, use empty string to not create any")
	createCmd.Flags().String("archive-name", "Archive", "Name of special mailbox for archive, use empty string to not create any")

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove USERNAME",
		Short: "Delete IMAP storage account",
		Long: `If IMAP connections are open and using the specified account,
messages access will be killed off immediately though connection will remain open. No cache
or other buffering takes effect.`,
		Args: cobra.ExactArgs(1),
		RunE: imapAcctRemove,
	}
	removeCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	removeCmd.Flags().BoolP("yes", "y", false, "Don't ask for confirmation")

	// Appendlimit subcommand
	appendlimitCmd := &cobra.Command{
		Use:   "appendlimit USERNAME",
		Short: "Query or set account's APPENDLIMIT value",
		Long: `APPENDLIMIT value determines the size of a message that
can be saved into a mailbox using IMAP APPEND command. This does not affect the size
of messages that can be delivered to the mailbox from non-IMAP sources (e.g. SMTP).

Global APPENDLIMIT value set via server configuration takes precedence over
per-account values configured using this command.

APPENDLIMIT value (either global or per-account) cannot be larger than
4 GiB due to IMAP protocol limitations.`,
		Args: cobra.ExactArgs(1),
		RunE: imapAcctAppendlimitCmd,
	}
	appendlimitCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	appendlimitCmd.Flags().IntP("value", "v", 0, "Set APPENDLIMIT to specified value (in bytes)")

	imapAcctCmd.AddCommand(listCmd, createCmd, removeCmd, appendlimitCmd)
	mailchatcli.AddSubcommand(imapAcctCmd)
}

func imapAcctList(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	mbe, ok := be.(module.ManageableStorage)
	if !ok {
		return fmt.Errorf("storage backend does not support accounts management using MailChat command")
	}

	list, err := mbe.ListIMAPAccts()
	if err != nil {
		return err
	}

	if len(list) == 0 {
		fmt.Fprintln(os.Stderr, "No users.")
	}

	for _, user := range list {
		fmt.Println(user)
	}
	return nil
}

func imapAcctCreate(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	mbe, ok := be.(module.ManageableStorage)
	if !ok {
		return fmt.Errorf("storage backend does not support accounts management using MailChat command")
	}

	username := args[0]

	if err := mbe.CreateIMAPAcct(username); err != nil {
		return err
	}

	act, err := mbe.GetIMAPAcct(username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	suu, ok := act.(SpecialUseUser)
	if !ok {
		fmt.Fprintf(os.Stderr, "Note: Storage backend does not support SPECIAL-USE IMAP extension")
	}

	noSpecialUse, _ := cmd.Flags().GetBool("no-specialuse")
	if noSpecialUse {
		return nil
	}

	createMbox := func(name, specialUseAttr string) error {
		if suu == nil {
			return act.CreateMailbox(name)
		}
		return suu.CreateMailboxSpecial(name, specialUseAttr)
	}

	if name, _ := cmd.Flags().GetString("sent-name"); name != "" {
		if err := createMbox(name, imap.SentAttr); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create sent folder: %v", err)
		}
	}
	if name, _ := cmd.Flags().GetString("trash-name"); name != "" {
		if err := createMbox(name, imap.TrashAttr); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create trash folder: %v", err)
		}
	}
	if name, _ := cmd.Flags().GetString("junk-name"); name != "" {
		if err := createMbox(name, imap.JunkAttr); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create junk folder: %v", err)
		}
	}
	if name, _ := cmd.Flags().GetString("drafts-name"); name != "" {
		if err := createMbox(name, imap.DraftsAttr); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create drafts folder: %v", err)
		}
	}
	if name, _ := cmd.Flags().GetString("archive-name"); name != "" {
		if err := createMbox(name, imap.ArchiveAttr); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create archive folder: %v", err)
		}
	}

	return nil
}

func imapAcctRemove(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	mbe, ok := be.(module.ManageableStorage)
	if !ok {
		return fmt.Errorf("storage backend does not support accounts management using MailChat command")
	}

	username := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		if !mailchatcli.Confirmation("Are you sure you want to delete this user account?", false) {
			return fmt.Errorf("cancelled")
		}
	}

	return mbe.DeleteIMAPAcct(username)
}

func imapAcctAppendlimitCmd(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	return imapAcctAppendlimit(be, cmd, args)
}