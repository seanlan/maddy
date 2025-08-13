package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

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
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
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
