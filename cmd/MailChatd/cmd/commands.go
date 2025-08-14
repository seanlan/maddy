package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	debugcmd "github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/dsoftgames/MailChat/app"
	mailchat "github.com/dsoftgames/MailChat"
	mailchatlog "github.com/dsoftgames/MailChat/framework/log"
	// Import for side-effect of registering CLI commands
	_ "github.com/dsoftgames/MailChat/internal/cli/ctl"
)

func initRootCmd(
	rootCmd *cobra.Command,
	txConfig client.TxConfig,
	basicManager module.BasicManager,
) {
	// Add MailChat mail server commands
	addMailChatCommands(rootCmd)
	
	rootCmd.AddCommand(
		NewInitCmd(basicManager, app.DefaultNodeHome),
		NewInPlaceTestnetCmd(),
		NewTestnetMultiNodeCmd(basicManager, banktypes.GenesisBalancesIterator{}),
		debugcmd.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
	)

	server.AddCommandsWithStartCmdOptions(rootCmd, app.DefaultNodeHome, newApp, appExport, server.StartCmdOptions{
		AddFlags: addModuleInitFlags,
	})

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome),
		queryCommand(),
		txCommand(),
		keys.Commands(),
	)
}

// addModuleInitFlags adds more flags to the start command.
func addModuleInitFlags(startCmd *cobra.Command) {
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.WaitTxCmd(),
		rpc.ValidatorCommand(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	return cmd
}

// addMailChatCommands adds MailChat mail server commands directly to the root command
func addMailChatCommands(rootCmd *cobra.Command) {
	// Add MailChat global flags to root command
	addMailChatGlobalFlags(rootCmd)

	// Add MailChat run command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the MailChat mail server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMailChatServer(cmd, args)
		},
	}
	// Add run command specific flags
	runCmd.Flags().String("libexec", mailchat.DefaultLibexecDirectory, "path to the libexec directory")
	runCmd.Flags().StringSlice("log", []string{"stderr"}, "default logging target(s)")
	runCmd.Flags().BoolP("v", "v", false, "print version and build metadata, then exit")
	runCmd.Flags().MarkHidden("v")

	// Add MailChat hash command
	hashCmd := &cobra.Command{
		Use:   "hash",
		Short: "Generate password hashes for use with pass_table",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHashCommand(cmd, args)
		},
	}
	hashCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin")
	hashCmd.Flags().String("hash", "bcrypt", "Use specified hash algorithm")
	hashCmd.Flags().Int("bcrypt-cost", 12, "Specify bcrypt cost value")
	hashCmd.Flags().Int("argon2-time", 3, "Time factor for Argon2id")
	hashCmd.Flags().Int("argon2-memory", 1024, "Memory in KiB to use for Argon2id")
	hashCmd.Flags().Int("argon2-threads", 1, "Threads to use for Argon2id")

	// Create creds command with subcommands
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

	// Add creds subcommands
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List user accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCredsCommand(cmd, args, "list")
		},
	}
	listCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	listCmd.Flags().Bool("quiet", false, "Do not print 'No users.' message")

	createCmd := &cobra.Command{
		Use:   "create USERNAME",
		Short: "Create user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCredsCommand(cmd, args, "create")
		},
	}
	createCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	createCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin")
	createCmd.Flags().String("hash", "bcrypt", "Hash algorithm to use")
	createCmd.Flags().Int("bcrypt-cost", 12, "Bcrypt cost value")

	removeCmd := &cobra.Command{
		Use:   "remove USERNAME",
		Short: "Delete user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCredsCommand(cmd, args, "remove")
		},
	}
	removeCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	removeCmd.Flags().BoolP("yes", "y", false, "Don't ask for confirmation")

	passwordCmd := &cobra.Command{
		Use:   "password USERNAME",
		Short: "Change user password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCredsCommand(cmd, args, "password")
		},
	}
	passwordCmd.Flags().String("cfg-block", "local_authdb", "Module configuration block to use")
	passwordCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin")

	credsCmd.AddCommand(listCmd, createCmd, removeCmd, passwordCmd)

	// Create IMAP commands
	imapAcctCmd := &cobra.Command{
		Use:   "imap-acct",
		Short: "IMAP storage accounts management",
		Long: `These subcommands can be used to list/create/delete IMAP storage
accounts for any storage backend supported by MailChat.`,
	}
	
	imapMboxesCmd := &cobra.Command{
		Use:   "imap-mboxes",
		Short: "IMAP mailboxes management",
		Long:  `These subcommands can be used to manage IMAP mailboxes.`,
	}

	imapMsgsCmd := &cobra.Command{
		Use:   "imap-msgs",
		Short: "IMAP messages management", 
		Long:  `These subcommands can be used to manage IMAP messages.`,
	}

	// Add all MailChat commands directly to root
	rootCmd.AddCommand(runCmd, hashCmd, credsCmd, imapAcctCmd, imapMboxesCmd, imapMsgsCmd)
}

// addMailChatGlobalFlags adds MailChat global flags to the command
func addMailChatGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("config", "", "Configuration file to use")
	cmd.PersistentFlags().Bool("debug", false, "Enable debug logging early")
}

// runMailChatServer runs the MailChat mail server
func runMailChatServer(cmd *cobra.Command, args []string) error {
	// Set up MailChat configuration
	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		// Set the config path for MailChat
		os.Setenv("MAILCHAT_CONFIG", configPath)
	}
	
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		mailchatlog.DefaultLogger.Debug = true
	}

	// Check for version flag
	showVersion, _ := cmd.Flags().GetBool("v")
	
	// Get log targets
	logTargets, _ := cmd.Flags().GetStringSlice("log")

	// Call MailChat's RunCobra function
	return mailchat.RunCobra(cmd, args, showVersion, logTargets)
}

// runHashCommand runs the MailChat hash command
func runHashCommand(cmd *cobra.Command, args []string) error {
	// This would need to import the actual hash functionality
	// For now, return an error indicating it needs implementation
	return fmt.Errorf("hash command implementation needs to be completed")
}

// runCredsCommand runs the MailChat credentials commands
func runCredsCommand(cmd *cobra.Command, args []string, action string) error {
	// This would need to import the actual credentials functionality
	// For now, return an error indicating it needs implementation
	return fmt.Errorf("creds %s command implementation needs to be completed", action)
}

// newApp creates the application
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	return app.New(
		logger, db, traceStore, true,
		appOpts,
		baseappOptions...,
	)
}

// appExport creates a new app (optionally at a given height) and exports state.
func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var bApp *app.App

	// this check is necessary as we use the flag in x/upgrade.
	// we can exit more gracefully by checking the flag here.
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	appOpts = viperAppOpts
	if height != -1 {
		bApp = app.New(logger, db, traceStore, false, appOpts)
		if err := bApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		bApp = app.New(logger, db, traceStore, true, appOpts)
	}

	return bApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// NewInitCmd creates a custom init command that also creates mailchat.conf
func NewInitCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {
	initCmd := genutilcli.InitCmd(mbm, defaultNodeHome)
	
	// Wrap the original RunE function
	originalRunE := initCmd.RunE
	initCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// First run the original init command
		if err := originalRunE(cmd, args); err != nil {
			return err
		}
		
		// Then create mailchat.conf in the node home directory
		return createMailChatConfig(cmd, args)
	}
	
	return initCmd
}

// createMailChatConfig creates a default mailchat.conf file
func createMailChatConfig(cmd *cobra.Command, args []string) error {
	// Get the node home directory
	nodeHome, err := cmd.Flags().GetString("home")
	if err != nil {
		return err
	}
	if nodeHome == "" {
		nodeHome = app.DefaultNodeHome
	}
	
	// Create mailchat.conf path
	configPath := filepath.Join(nodeHome, "mailchat.conf")
	
	// Check if mailchat.conf already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("MailChat configuration file already exists at %s\n", configPath)
		return nil
	}
	
	// Create the default mailchat.conf content based on the existing template
	configContent := `## MailChat - default configuration file (2022-06-18)
# Suitable for small-scale deployments. Uses its own format for local users DB,
# should be managed via mailchatd subcommands.
#
# See tutorials at https://maddy.email for guidance on typical
# configuration changes.

# ----------------------------------------------------------------------------
# Base variables

$(hostname) = example.com
$(primary_domain) = example.com
$(local_domains) = $(primary_domain)

# tls file certs/$(hostname)/fullchain.pem certs/$(hostname)/privkey.pem

tls {
    loader acme {
        hostname $(hostname)
        email postmaster@$(hostname)
        agreed
        challenge dns-01
        dns cloudflare {
            api_token YOUR_CLOUDFLARE_API_TOKEN_HERE
        }
    }
}

# ----------------------------------------------------------------------------
# blockchains
blockchain.ethereum amoy {
    chain_id 80002
    rpc_url https://polygon-amoy.gateway.tenderly.co
}

# ----------------------------------------------------------------------------
# Local storage & authentication

# imapsql module stores all indexes and metadata necessary for IMAP using a
# relational database. It is used by IMAP endpoint for mailbox access and
# also by SMTP & Submission endpoints for delivery of local messages.
#
# IMAP accounts, mailboxes and all message metadata can be inspected using
# imap-* subcommands of mailchatd.

storage.imapsql local_mailboxes {
    driver sqlite3
    dsn imapsql.db
}

# pass_table provides local hashed passwords storage for authentication of
# users. It can be configured to use any "table" module, in default
# configuration a table in SQLite DB is used.
# Table can be replaced to use e.g. a file for passwords. Or pass_table module
# can be replaced altogether to use some external source of credentials (e.g.
# PAM, /etc/shadow file).
#
# If table module supports it (sql_table does) - credentials can be managed
# using 'mailchatd creds' command.

# auth.pass_table local_authdb {
#     table sql_table {
#         driver sqlite3
#         dsn credentials.db
#         table_name passwords
#     }
# }

# pass blockchain module provides authentication using blockchain wallets.
auth.pass_blockchain blockchain_atuh {
    blockchain &amoy
    storage &local_mailboxes
}

# ----------------------------------------------------------------------------
# SMTP endpoints + message routing

hostname $(hostname)

table.chain local_rewrites {
    optional_step regexp "(.+)\+(.+)@(.+)" "$1@$3"
    optional_step static {
        entry postmaster postmaster@$(primary_domain)
    }
    optional_step file ~/.mailcoin/aliases
}

msgpipeline local_routing {
    # Insert handling for special-purpose local domains here.
    # e.g.
    # destination lists.example.org {
    #     deliver_to lmtp tcp://127.0.0.1:8024
    # }

    destination postmaster $(local_domains) {
        modify {
            replace_rcpt &local_rewrites
            blockchain_tx &amoy
        }

        deliver_to &local_mailboxes
    }

    default_destination {
        reject 550 5.1.1 "User doesn't exist"
    }
}

smtp tcp://0.0.0.0:8825 {
    limits {
        # Up to 20 msgs/sec across max. 10 SMTP connections.
        all rate 20 1s
        all concurrency 10
    }

    dmarc yes
    check {
        require_mx_record
        dkim
        spf
    }

    source $(local_domains) {
        reject 501 5.1.8 "Use Submission for outgoing SMTP"
    }
    default_source {
        destination postmaster $(local_domains) {
            deliver_to &local_routing
        }
        default_destination {
            reject 550 5.1.1 "User doesn't exist"
        }
    }
}

submission tls://0.0.0.0:465 tcp://0.0.0.0:587 {
    limits {
        # Up to 50 msgs/sec across any amount of SMTP connections.
        all rate 50 1s
    }

    auth &blockchain_atuh

    source $(local_domains) {
        check {
            authorize_sender {
                prepare_email &local_rewrites
                user_to_email identity
            }
        }

        modify {
            blockchain_tx &amoy
        }

        destination postmaster $(local_domains) {
            deliver_to &local_routing
        }
        default_destination {
            modify {
                dkim $(primary_domain) $(local_domains) default
            }
            deliver_to &remote_queue
        }
    }
    default_source {
        reject 501 5.1.8 "Non-local sender domain"
    }
}

target.remote outbound_delivery {
    limits {
        # Up to 20 msgs/sec across max. 10 SMTP connections
        # for each recipient domain.
        destination rate 20 1s
        destination concurrency 10
    }
    mx_auth {
        dane {
            # SMTP port for DANE TLSA record queries (should match smtp_port above)
            # Uncomment and modify if using custom port
            smtp_port 8825
        }
        mtasts {
            cache fs
            fs_dir mtasts_cache/
        }
        local_policy {
            min_tls_level encrypted
            min_mx_level none
        }
    }
}

target.queue remote_queue {
    target &outbound_delivery

    autogenerated_msg_domain $(primary_domain)
    bounce {
        destination postmaster $(local_domains) {
            deliver_to &local_routing
        }
        default_destination {
            reject 550 5.0.0 "Refusing to send DSNs to non-local addresses"
        }
    }
}

# ----------------------------------------------------------------------------
# IMAP endpoints

imap tls://0.0.0.0:993 tcp://0.0.0.0:143 {
    auth &blockchain_atuh
    storage &local_mailboxes
}
`

	// Write the config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create mailchat.conf: %w", err)
	}
	
	fmt.Printf("Created MailChat configuration file at %s\n", configPath)
	return nil
}
