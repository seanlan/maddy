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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/dsoftgames/MailChat"
	parser "github.com/dsoftgames/MailChat/framework/cfgparser"
	"github.com/dsoftgames/MailChat/framework/config"
	"github.com/dsoftgames/MailChat/framework/hooks"
	"github.com/dsoftgames/MailChat/framework/module"
	"github.com/dsoftgames/MailChat/internal/updatepipe"
)

func closeIfNeeded(i interface{}) {
	if c, ok := i.(io.Closer); ok {
		c.Close()
	}
}

func getCfgBlockModule(cmd *cobra.Command) (map[string]interface{}, *mailchat.ModInfo, error) {
	cfgPath, _ := cmd.Flags().GetString("config")
	if cfgPath == "" {
		return nil, nil, fmt.Errorf("config is required")
	}
	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer cfgFile.Close()
	cfgNodes, err := parser.Read(cfgFile, cfgFile.Name())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	globals, cfgNodes, err := mailchat.ReadGlobals(cfgNodes)
	if err != nil {
		return nil, nil, err
	}

	if err := mailchat.InitDirs(); err != nil {
		return nil, nil, err
	}

	module.NoRun = true
	_, mods, err := mailchat.RegisterModules(globals, cfgNodes)
	if err != nil {
		return nil, nil, err
	}
	defer hooks.RunHooks(hooks.EventShutdown)

	cfgBlock, _ := cmd.Flags().GetString("cfg-block")
	if cfgBlock == "" {
		return nil, nil, fmt.Errorf("cfg-block is required")
	}
	var mod mailchat.ModInfo
	for _, m := range mods {
		if m.Instance.InstanceName() == cfgBlock {
			mod = m
			break
		}
	}
	if mod.Instance == nil {
		return nil, nil, fmt.Errorf("unknown configuration block: %s", cfgBlock)
	}

	return globals, &mod, nil
}

func openStorage(cmd *cobra.Command) (module.Storage, error) {
	globals, mod, err := getCfgBlockModule(cmd)
	if err != nil {
		return nil, err
	}

	storage, ok := mod.Instance.(module.Storage)
	if !ok {
		cfgBlock, _ := cmd.Flags().GetString("cfg-block")
		return nil, fmt.Errorf("configuration block %s is not an IMAP storage", cfgBlock)
	}

	if err := mod.Instance.Init(config.NewMap(globals, mod.Cfg)); err != nil {
		return nil, fmt.Errorf("Error: module initialization failed: %w", err)
	}

	if updStore, ok := mod.Instance.(updatepipe.Backend); ok {
		if err := updStore.EnableUpdatePipe(updatepipe.ModePush); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Failed to initialize update pipe, do not remove messages from mailboxes open by clients: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "No update pipe support, do not remove messages from mailboxes open by clients\n")
	}

	return storage, nil
}

func openUserDB(cmd *cobra.Command) (module.PlainUserDB, error) {
	globals, mod, err := getCfgBlockModule(cmd)
	if err != nil {
		return nil, err
	}

	userDB, ok := mod.Instance.(module.PlainUserDB)
	if !ok {
		cfgBlock, _ := cmd.Flags().GetString("cfg-block")
		return nil, fmt.Errorf("configuration block %s is not a local credentials store", cfgBlock)
	}

	if err := mod.Instance.Init(config.NewMap(globals, mod.Cfg)); err != nil {
		return nil, fmt.Errorf("Error: module initialization failed: %w", err)
	}

	return userDB, nil
}
