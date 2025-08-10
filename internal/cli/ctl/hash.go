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
	"mailcoin/internal/auth/pass_table"
	maddycli "mailcoin/internal/cli"
	clitools2 "mailcoin/internal/cli/clitools"
)

func init() {
	hashCmd := &cobra.Command{
		Use:   "hash",
		Short: "Generate password hashes for use with pass_table",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := maddycli.NewCobraContext(cmd, args)
			return hashCommand(ctx)
		},
	}

	hashCmd.Flags().StringP("password", "p", "", "Use PASSWORD instead of reading password from stdin\n\t\tWARNING: Provided only for debugging convenience. Don't leave your passwords in shell history!")
	hashCmd.Flags().String("hash", "bcrypt", "Use specified hash algorithm")
	hashCmd.Flags().Int("bcrypt-cost", bcrypt.DefaultCost, "Specify bcrypt cost value")
	hashCmd.Flags().Int("argon2-time", 3, "Time factor for Argon2id")
	hashCmd.Flags().Int("argon2-memory", 1024, "Memory in KiB to use for Argon2id")
	hashCmd.Flags().Int("argon2-threads", 1, "Threads to use for Argon2id")

	maddycli.AddSubcommand(hashCmd)
}

func hashCommand(ctx *maddycli.CobraContext) error {
	hashFunc := ctx.String("hash")
	if hashFunc == "" {
		hashFunc = pass_table.DefaultHash
	}

	hashCompute := pass_table.HashCompute[hashFunc]
	if hashCompute == nil {
		var funcs []string
		for k := range pass_table.HashCompute {
			funcs = append(funcs, k)
		}

		return fmt.Errorf("Error: Unknown hash function, available: %s", strings.Join(funcs, ", "))
	}

	opts := pass_table.HashOpts{
		BcryptCost:    bcrypt.DefaultCost,
		Argon2Memory:  1024,
		Argon2Time:    2,
		Argon2Threads: 1,
	}
	if ctx.IsSet("bcrypt-cost") {
		if ctx.Int("bcrypt-cost") > bcrypt.MaxCost {
			return fmt.Errorf("Error: too big bcrypt cost")
		}
		if ctx.Int("bcrypt-cost") < bcrypt.MinCost {
			return fmt.Errorf("Error: too small bcrypt cost")
		}
		opts.BcryptCost = ctx.Int("bcrypt-cost")
	}
	if ctx.IsSet("argon2-memory") {
		opts.Argon2Memory = uint32(ctx.Int("argon2-memory"))
	}
	if ctx.IsSet("argon2-time") {
		opts.Argon2Time = uint32(ctx.Int("argon2-time"))
	}
	if ctx.IsSet("argon2-threads") {
		opts.Argon2Threads = uint8(ctx.Int("argon2-threads"))
	}

	var pass string
	if ctx.IsSet("password") {
		pass = ctx.String("password")
	} else {
		var err error
		pass, err = clitools2.ReadPassword("Password")
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
