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

package mailcoin

import (
	"errors"
	"fmt"
	"io"
	mailcoincli "mailcoin/internal/cli"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/caddyserver/certmagic"
	"github.com/spf13/cobra"
	parser "mailcoin/framework/cfgparser"
	"mailcoin/framework/config"
	modconfig "mailcoin/framework/config/module"
	"mailcoin/framework/config/tls"
	"mailcoin/framework/hooks"
	"mailcoin/framework/log"
	"mailcoin/framework/module"
	"mailcoin/internal/authz"
	// Import packages for side-effect of module registration.
	_ "mailcoin/internal/auth/dovecot_sasl"
	_ "mailcoin/internal/auth/external"
	_ "mailcoin/internal/auth/ldap"
	_ "mailcoin/internal/auth/netauth"
	_ "mailcoin/internal/auth/pam"
	_ "mailcoin/internal/auth/pass_blockchain"
	_ "mailcoin/internal/auth/pass_table"
	_ "mailcoin/internal/auth/plain_separate"
	_ "mailcoin/internal/auth/shadow"
	_ "mailcoin/internal/blockchain"
	_ "mailcoin/internal/check/authorize_sender"
	_ "mailcoin/internal/check/command"
	_ "mailcoin/internal/check/dkim"
	_ "mailcoin/internal/check/dns"
	_ "mailcoin/internal/check/dnsbl"
	_ "mailcoin/internal/check/milter"
	_ "mailcoin/internal/check/requiretls"
	_ "mailcoin/internal/check/rspamd"
	_ "mailcoin/internal/check/spf"
	_ "mailcoin/internal/endpoint/dovecot_sasld"
	_ "mailcoin/internal/endpoint/imap"
	_ "mailcoin/internal/endpoint/openmetrics"
	_ "mailcoin/internal/endpoint/smtp"
	_ "mailcoin/internal/imap_filter"
	_ "mailcoin/internal/imap_filter/command"
	_ "mailcoin/internal/libdns"
	_ "mailcoin/internal/modify"
	_ "mailcoin/internal/modify/dkim"
	_ "mailcoin/internal/storage/blob/fs"
	_ "mailcoin/internal/storage/blob/s3"
	_ "mailcoin/internal/storage/imapsql"
	_ "mailcoin/internal/table"
	_ "mailcoin/internal/target/queue"
	_ "mailcoin/internal/target/remote"
	_ "mailcoin/internal/target/smtp"
	_ "mailcoin/internal/tls"
	_ "mailcoin/internal/tls/acme"
)

var (
	Version = "go-build"

	enableDebugFlags = false
)

func BuildInfo() string {
	version := Version
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	return fmt.Sprintf(`%s %s/%s %s

default config: %s
default state_dir: %s
default runtime_dir: %s`,
		version, runtime.GOOS, runtime.GOARCH, runtime.Version(),
		filepath.Join(ConfigDirectory, "mailcoin.conf"),
		DefaultStateDirectory,
		DefaultRuntimeDirectory)
}

var (
	configPath string
)

func init() {
	configPath = filepath.Join(ConfigDirectory, "mailcoin.conf")
	mailcoincli.AddGlobalStringFlag("config", "Configuration file to use", "MADDY_CONFIG", configPath, &configPath)
	mailcoincli.AddGlobalBoolFlag("debug", "enable debug logging early", &log.DefaultLogger.Debug)
	var (
		logTargets []string
		showVersion bool
		debugPprof string
		debugBlockProfRate int
		debugMutexProfFract int
	)
	config.LibexecDirectory = DefaultLibexecDirectory

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCobra(cmd, args, showVersion, logTargets)
		},
	}

	runCmd.Flags().StringVar(&config.LibexecDirectory, "libexec", DefaultLibexecDirectory, "path to the libexec directory")
	runCmd.Flags().StringSliceVar(&logTargets, "log", []string{"stderr"}, "default logging target(s)")
	runCmd.Flags().BoolVarP(&showVersion, "v", "v", false, "print version and build metadata, then exit")
	runCmd.Flags().MarkHidden("v")

	if enableDebugFlags {
		runCmd.Flags().StringVar(&debugPprof, "debug.pprof", "", "enable live profiler HTTP endpoint and listen on the specified address")
		runCmd.Flags().IntVar(&debugBlockProfRate, "debug.blockprofrate", 0, "set blocking profile rate")
		runCmd.Flags().IntVar(&debugMutexProfFract, "debug.mutexproffract", 0, "set mutex profile fraction")
	}

	mailcoincli.AddSubcommand(runCmd)
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version and build metadata, then exit",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(BuildInfo())
			return nil
		},
	}

	mailcoincli.AddSubcommand(versionCmd)

	if enableDebugFlags {
		// Debug flags will be added to the run command specifically
		// since they're only used during server execution
	}
}

// Run is the entry point for all server-running code. It takes care of command line arguments processing,
// logging initialization, directives setup, configuration reading. After all that, it
// calls moduleMain to initialize and run modules.
// Run is the legacy entry point for urfave/cli compatibility
func Run(c interface{}) error {
	// This function is kept for backwards compatibility but should not be used
	return fmt.Errorf("legacy Run function called - use RunCobra instead")
}

// RunCobra is the entry point for all server-running code with Cobra.
func RunCobra(cmd *cobra.Command, args []string, showVersion bool, logTargets []string) error {
	certmagic.UserAgent = "mailcoin/" + Version

	if len(args) != 0 {
		return fmt.Errorf("usage: %s [options]", os.Args[0])
	}

	if showVersion {
		fmt.Println("mailcoin", BuildInfo())
		return nil
	}

	var err error
	log.DefaultLogger.Out, err = LogOutputOption(logTargets)
	if err != nil {
		systemdStatusErr(err)
		return fmt.Errorf("%s", err.Error())
	}

	initDebugCobra(cmd)

	os.Setenv("PATH", config.LibexecDirectory+string(filepath.ListSeparator)+os.Getenv("PATH"))

	log.Printf("Starting mailcoin %s \n", Version)
	f, err := os.Open(configPath)
	if err != nil {
		systemdStatusErr(err)
		return fmt.Errorf("%s", err.Error())
	}
	defer f.Close()

	cfg, err := parser.Read(f, configPath)
	if err != nil {
		systemdStatusErr(err)
		return fmt.Errorf("%s", err.Error())
	}

	defer log.DefaultLogger.Out.Close()

	if err := moduleMain(cfg); err != nil {
		systemdStatusErr(err)
		return fmt.Errorf("%s", err.Error())
	}

	return nil
}

func initDebugCobra(cmd *cobra.Command) {
	if !enableDebugFlags {
		return
	}

	if cmd.Flags().Changed("debug.pprof") {
		profileEndpoint, _ := cmd.Flags().GetString("debug.pprof")
		go func() {
			log.Println("listening on", "http://"+profileEndpoint, "for profiler requests")
			log.Println("failed to listen on profiler endpoint:", http.ListenAndServe(profileEndpoint, nil))
		}()
	}

	// These values can also be affected by environment so set them
	// only if argument is specified.
	if cmd.Flags().Changed("debug.mutexproffract") {
		mutexProfFract, _ := cmd.Flags().GetInt("debug.mutexproffract")
		runtime.SetMutexProfileFraction(mutexProfFract)
	}
	if cmd.Flags().Changed("debug.blockprofrate") {
		blockProfRate, _ := cmd.Flags().GetInt("debug.blockprofrate")
		runtime.SetBlockProfileRate(blockProfRate)
	}
}

func InitDirs() error {
	if config.StateDirectory == "" {
		config.StateDirectory = DefaultStateDirectory
	}
	if config.RuntimeDirectory == "" {
		config.RuntimeDirectory = DefaultRuntimeDirectory
	}
	if config.LibexecDirectory == "" {
		config.LibexecDirectory = DefaultLibexecDirectory
	}

	if err := ensureDirectoryWritable(config.StateDirectory); err != nil {
		return err
	}
	if err := ensureDirectoryWritable(config.RuntimeDirectory); err != nil {
		return err
	}

	// Make sure all paths we are going to use are absolute
	// before we change the working directory.
	if !filepath.IsAbs(config.StateDirectory) {
		return errors.New("statedir should be absolute")
	}
	if !filepath.IsAbs(config.RuntimeDirectory) {
		return errors.New("runtimedir should be absolute")
	}
	if !filepath.IsAbs(config.LibexecDirectory) {
		return errors.New("-libexec should be absolute")
	}

	// Change the working directory to make all relative paths
	// in configuration relative to state directory.
	if err := os.Chdir(config.StateDirectory); err != nil {
		log.Println(err)
	}

	return nil
}

func ensureDirectoryWritable(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}

	testFile, err := os.Create(filepath.Join(path, "writeable-test"))
	if err != nil {
		return err
	}
	testFile.Close()
	return os.RemoveAll(testFile.Name())
}

func ReadGlobals(cfg []config.Node) (map[string]interface{}, []config.Node, error) {
	// don't know what caused the inability to set config Default value of StateDirectory， so I set it here
	config.StateDirectory = DefaultStateDirectory
	globals := config.NewMap(nil, config.Node{Children: cfg})
	globals.String("state_dir", false, false, DefaultStateDirectory, &config.StateDirectory)
	globals.String("runtime_dir", false, false, DefaultRuntimeDirectory, &config.RuntimeDirectory)
	globals.String("hostname", false, false, "", nil)
	globals.String("autogenerated_msg_domain", false, false, "", nil)
	globals.Custom("tls", false, false, nil, tls.TLSDirective, nil)
	globals.Custom("tls_client", false, false, nil, tls.TLSClientBlock, nil)
	globals.Bool("storage_perdomain", false, false, nil)
	globals.Bool("auth_perdomain", false, false, nil)
	globals.StringList("auth_domains", false, false, nil, nil)
	globals.Custom("log", false, false, defaultLogOutput, logOutput, &log.DefaultLogger.Out)
	globals.Bool("debug", false, log.DefaultLogger.Debug, &log.DefaultLogger.Debug)
	config.EnumMapped(globals, "auth_map_normalize", true, false, authz.NormalizeFuncs, authz.NormalizeAuto, nil)
	modconfig.Table(globals, "auth_map", true, false, nil, nil)
	globals.AllowUnknown()
	unknown, err := globals.Process()
	return globals.Values, unknown, err
}

func moduleMain(cfg []config.Node) error {
	globals, modBlocks, err := ReadGlobals(cfg)
	fmt.Printf("config.StateDirectory: %v\n", config.StateDirectory)
	if err != nil {
		return err
	}

	if err := InitDirs(); err != nil {
		return err
	}

	hooks.AddHook(hooks.EventLogRotate, reinitLogging)

	endpoints, mods, err := RegisterModules(globals, modBlocks)
	if err != nil {
		return err
	}

	err = initModules(globals, endpoints, mods)
	if err != nil {
		return err
	}

	systemdStatus(SDReady, "Listening for incoming connections...")

	handleSignals()

	systemdStatus(SDStopping, "Waiting for running transactions to complete...")

	hooks.RunHooks(hooks.EventShutdown)

	return nil
}

type ModInfo struct {
	Instance module.Module
	Cfg      config.Node
}

func RegisterModules(globals map[string]interface{}, nodes []config.Node) (endpoints, mods []ModInfo, err error) {
	mods = make([]ModInfo, 0, len(nodes))

	for _, block := range nodes {
		var instName string
		var modAliases []string
		if len(block.Args) == 0 {
			instName = block.Name
		} else {
			instName = block.Args[0]
			modAliases = block.Args[1:]
		}

		modName := block.Name

		endpFactory := module.GetEndpoint(modName)
		if endpFactory != nil {
			inst, err := endpFactory(modName, block.Args)
			if err != nil {
				return nil, nil, err
			}

			endpoints = append(endpoints, ModInfo{Instance: inst, Cfg: block})
			continue
		}

		factory := module.Get(modName)
		if factory == nil {
			return nil, nil, config.NodeErr(block, "unknown module or global directive: %s", modName)
		}

		if module.HasInstance(instName) {
			return nil, nil, config.NodeErr(block, "config block named %s already exists", instName)
		}

		inst, err := factory(modName, instName, modAliases, nil)
		if err != nil {
			return nil, nil, err
		}

		module.RegisterInstance(inst, config.NewMap(globals, block))
		for _, alias := range modAliases {
			if module.HasInstance(alias) {
				return nil, nil, config.NodeErr(block, "config block named %s already exists", alias)
			}
			module.RegisterAlias(alias, instName)
		}

		log.Debugf("%v:%v: register config block %v %v", block.File, block.Line, instName, modAliases)
		mods = append(mods, ModInfo{Instance: inst, Cfg: block})
	}

	if len(endpoints) == 0 {
		return nil, nil, fmt.Errorf("at least one endpoint should be configured")
	}

	return endpoints, mods, nil
}

func initModules(globals map[string]interface{}, endpoints, mods []ModInfo) error {
	for _, endp := range endpoints {
		if err := endp.Instance.Init(config.NewMap(globals, endp.Cfg)); err != nil {
			return err
		}

		if closer, ok := endp.Instance.(io.Closer); ok {
			endp := endp
			hooks.AddHook(hooks.EventShutdown, func() {
				log.Debugf("close %s (%s)", endp.Instance.Name(), endp.Instance.InstanceName())
				if err := closer.Close(); err != nil {
					log.Printf("module %s (%s) close failed: %v", endp.Instance.Name(), endp.Instance.InstanceName(), err)
				}
			})
		}
	}

	for _, inst := range mods {
		if module.Initialized[inst.Instance.InstanceName()] {
			continue
		}

		return fmt.Errorf("Unused configuration block at %s:%d - %s (%s)",
			inst.Cfg.File, inst.Cfg.Line, inst.Instance.InstanceName(), inst.Instance.Name())
	}

	return nil
}
