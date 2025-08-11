package ctl

// All legacy CLI commands have been successfully migrated to Cobra framework.
// The following commands are now available:
// - creds (user credentials management) - implemented in creds.go
// - imap-acct (IMAP account management) - implemented in imap_acct.go  
// - imap-mboxes (IMAP mailbox management) - implemented in imap_mboxes.go
// - imap-msgs (IMAP message management) - implemented in imap_msgs.go
//
// Migration completed - all commands now use Cobra instead of urfave/cli
func init() {
	// Migration complete - no legacy commands remain
}