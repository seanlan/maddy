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
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"github.com/dsoftgames/MailChat/internal/auth/pass_table"
	mailchatcli "github.com/dsoftgames/MailChat/internal/cli"
)

func init() {
	hashCmd := &cobra.Command{
		Use:   "hash",
		Short: "Generate password hashes for use with pass_table",
		RunE:  hashCommand,
	}

	hashCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin\n\t\tWARNING: Provided only for debugging convenience. Don't leave your passwords in shell history!")
	hashCmd.Flags().String("hash", "bcrypt", "Use specified hash algorithm")
	hashCmd.Flags().Int("bcrypt-cost", bcrypt.DefaultCost, "Specify bcrypt cost value")
	hashCmd.Flags().Int("argon2-time", 3, "Time factor for Argon2id")
	hashCmd.Flags().Int("argon2-memory", 1024, "Memory in KiB to use for Argon2id")
	hashCmd.Flags().Int("argon2-threads", 1, "Threads to use for Argon2id")

	mailchatcli.AddSubcommand(hashCmd)
}

func hashCommand(cmd *cobra.Command, args []string) error {
	hashFunc, _ := cmd.Flags().GetString("hash")
	if hashFunc == "" {
		hashFunc = pass_table.DefaultHash
	}

	hashCompute := pass_table.HashCompute[hashFunc]
	if hashCompute == nil {
		var funcs []string
		for k := range pass_table.HashCompute {
			funcs = append(funcs, k)
		}

		return fmt.Errorf("unknown hash function, available: %s", strings.Join(funcs, ", "))
	}

	opts := pass_table.HashOpts{
		BcryptCost:    bcrypt.DefaultCost,
		Argon2Memory:  1024,
		Argon2Time:    2,
		Argon2Threads: 1,
	}
	if cmd.Flags().Changed("bcrypt-cost") {
		bcryptCost, _ := cmd.Flags().GetInt("bcrypt-cost")
		if bcryptCost > bcrypt.MaxCost {
			return fmt.Errorf("bcrypt cost %d exceeds maximum %d", bcryptCost, bcrypt.MaxCost)
		}
		if bcryptCost < bcrypt.MinCost {
			return fmt.Errorf("bcrypt cost %d is below minimum %d", bcryptCost, bcrypt.MinCost)
		}
		opts.BcryptCost = bcryptCost
	}
	if cmd.Flags().Changed("argon2-memory") {
		argon2Memory, _ := cmd.Flags().GetInt("argon2-memory")
		opts.Argon2Memory = uint32(argon2Memory)
	}
	if cmd.Flags().Changed("argon2-time") {
		argon2Time, _ := cmd.Flags().GetInt("argon2-time")
		opts.Argon2Time = uint32(argon2Time)
	}
	if cmd.Flags().Changed("argon2-threads") {
		argon2Threads, _ := cmd.Flags().GetInt("argon2-threads")
		opts.Argon2Threads = uint8(argon2Threads)
	}

	var pass string
	if cmd.Flags().Changed("password") {
		pass, _ = cmd.Flags().GetString("password")
	} else {
		var err error
		pass, err = mailchatcli.ReadPassword("Password")
		if err != nil {
			return err
		}
	}

	if pass == "" {
		fmt.Fprintln(os.Stderr, "WARNING: This is the hash of an empty string")
	}
	if strings.TrimSpace(pass) != pass {
		fmt.Fprintln(os.Stderr, "WARNING: There is leading/trailing whitespace in the string")
	}

	hash, err := hashCompute(opts, pass)
	if err != nil {
		return err
	}
	fmt.Println(hashFunc + ":" + hash)
	return nil
}
