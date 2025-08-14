/*
MailChat - Composable all-in-one email server.
Copyright © 2019-2020 Max Mazurov <fox.cpp@disroot.org>, MailChat contributors

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
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/emersion/go-imap"
	imapsql "github.com/foxcpp/go-imap-sql"
	"github.com/spf13/cobra"
	mailchatcli "github.com/dsoftgames/MailChat/internal/cli"
)

func init() {
	imapMsgsCmd := &cobra.Command{
		Use:   "imap-msgs",
		Short: "IMAP messages management",
	}

	// Add subcommand
	addCmd := &cobra.Command{
		Use:   "add USERNAME MAILBOX",
		Short: "Add message to mailbox",
		Long:  "Reads message body (with headers) from stdin. Prints UID of created message on success.",
		Args:  cobra.ExactArgs(2),
		RunE:  msgsAdd,
	}
	addCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	addCmd.Flags().StringSliceP("flag", "f", nil, "Add flag to message. Can be specified multiple times")
	addCmd.Flags().StringP("date", "d", "", "Set internal date value to specified one in ISO 8601 format (2006-01-02T15:04:05Z07:00)")

	// Add-flags subcommand
	addFlagsCmd := &cobra.Command{
		Use:   "add-flags USERNAME MAILBOX SEQSET FLAG...",
		Short: "Add flags to messages",
		Args:  cobra.MinimumNArgs(4),
		RunE:  msgsAddFlags,
	}
	addFlagsCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	addFlagsCmd.Flags().Bool("uid", false, "Use UID STORE instead of STORE")

	// Remove-flags subcommand
	removeFlagsCmd := &cobra.Command{
		Use:   "remove-flags USERNAME MAILBOX SEQSET FLAG...",
		Short: "Remove flags from messages",
		Args:  cobra.MinimumNArgs(4),
		RunE:  msgsRemoveFlags,
	}
	removeFlagsCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	removeFlagsCmd.Flags().Bool("uid", false, "Use UID STORE instead of STORE")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list USERNAME MAILBOX [SEQSET]",
		Short: "List messages in mailbox",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  msgsList,
	}
	listCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	listCmd.Flags().Bool("uid", false, "Use UIDs instead of sequence numbers")

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove USERNAME MAILBOX SEQSET",
		Short: "Delete messages from mailbox",
		Args:  cobra.ExactArgs(3),
		RunE:  msgsRemove,
	}
	removeCmd.Flags().String("cfg-block", "local_mailboxes", "Module configuration block to use")
	removeCmd.Flags().Bool("uid", false, "Use UIDs instead of sequence numbers")
	removeCmd.Flags().BoolP("yes", "y", false, "Don't ask for confirmation")

	imapMsgsCmd.AddCommand(addCmd, addFlagsCmd, removeFlagsCmd, listCmd, removeCmd)
	mailchatcli.AddSubcommand(imapMsgsCmd)
}

func msgsAdd(cmd *cobra.Command, args []string) error {
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

	flags, _ := cmd.Flags().GetStringSlice("flag")
	if flags == nil {
		flags = []string{}
	}

	date := time.Now()
	if cmd.Flags().Changed("date") {
		dateStr, _ := cmd.Flags().GetString("date")
		parsedDate, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
		date = parsedDate
	}

	buf := bytes.Buffer{}
	if _, err := io.Copy(&buf, os.Stdin); err != nil {
		return err
	}

	if buf.Len() == 0 {
		return fmt.Errorf("empty message, refusing to continue")
	}

	status, err := u.Status(name, []imap.StatusItem{imap.StatusUidNext})
	if err != nil {
		return err
	}

	if err := u.CreateMessage(name, flags, date, &buf, nil); err != nil {
		return err
	}

	// TODO: Use APPENDUID
	fmt.Println(status.UidNext)

	return nil
}

func msgsAddFlags(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	mailbox := args[1]
	seqsetStr := args[2]
	flags := args[3:]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	seq, err := imap.ParseSeqSet(seqsetStr)
	if err != nil {
		return err
	}

	_, mbox, err := u.GetMailbox(mailbox, true, nil)
	if err != nil {
		return err
	}

	useUID, _ := cmd.Flags().GetBool("uid")
	mboxB := mbox.(*imapsql.Mailbox)

	return mboxB.UpdateMessagesFlags(useUID, seq, imap.AddFlags, false, flags)
}

func msgsRemoveFlags(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	mailbox := args[1]
	seqsetStr := args[2]
	flags := args[3:]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	seq, err := imap.ParseSeqSet(seqsetStr)
	if err != nil {
		return err
	}

	_, mbox, err := u.GetMailbox(mailbox, true, nil)
	if err != nil {
		return err
	}

	useUID, _ := cmd.Flags().GetBool("uid")
	mboxB := mbox.(*imapsql.Mailbox)

	return mboxB.UpdateMessagesFlags(useUID, seq, imap.RemoveFlags, false, flags)
}

func msgsList(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	mailbox := args[1]

	var seqsetStr string
	if len(args) >= 3 {
		seqsetStr = args[2]
	} else {
		seqsetStr = "1:*"
	}

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	seq, err := imap.ParseSeqSet(seqsetStr)
	if err != nil {
		return err
	}

	_, mbox, err := u.GetMailbox(mailbox, false, nil)
	if err != nil {
		return err
	}

	useUID, _ := cmd.Flags().GetBool("uid")
	items := []imap.FetchItem{imap.FetchFlags, imap.FetchInternalDate}
	if useUID {
		items = append(items, imap.FetchUid)
	}

	ch := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- mbox.ListMessages(useUID, seq, items, ch)
	}()

	for msg := range ch {
		if useUID {
			fmt.Printf("UID %d: ", msg.Uid)
		} else {
			fmt.Printf("SeqNum %d: ", msg.SeqNum)
		}
		fmt.Printf("Flags: %v, Date: %s\n", msg.Flags, msg.InternalDate.Format(time.RFC3339))
	}

	return <-done
}

func msgsRemove(cmd *cobra.Command, args []string) error {
	be, err := openStorage(cmd)
	if err != nil {
		return err
	}
	defer closeIfNeeded(be)

	username := args[0]
	name := args[1]
	seqset := args[2]

	useUID, _ := cmd.Flags().GetBool("uid")
	if !useUID {
		fmt.Fprintln(os.Stderr, "WARNING: --uid=true will be the default in future versions")
	}

	seq, err := imap.ParseSeqSet(seqset)
	if err != nil {
		return err
	}

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}

	_, mbox, err := u.GetMailbox(name, true, nil)
	if err != nil {
		return err
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		if !mailchatcli.Confirmation("Are you sure you want to delete these messages?", false) {
			return fmt.Errorf("cancelled")
		}
	}

	mboxB := mbox.(*imapsql.Mailbox)
	return mboxB.DelMessages(useUID, seq)
}