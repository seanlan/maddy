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

	imapbackend "github.com/emersion/go-imap/backend"
	"github.com/spf13/cobra"
	"github.com/dsoftgames/MailChat/framework/module"
)

// Copied from go-imap-backend-tests.

// AppendLimitUser is extension for backend.User interface which allows to
// set append limit value for testing and administration purposes.
type AppendLimitUser interface {
	imapbackend.AppendLimitUser

	// SetMessageLimit sets new value for limit.
	// nil pointer means no limit.
	SetMessageLimit(val *uint32) error
}

func imapAcctAppendlimit(be module.Storage, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("USERNAME is required")
	}
	username := args[0]

	u, err := be.GetIMAPAcct(username)
	if err != nil {
		return err
	}
	userAL, ok := u.(AppendLimitUser)
	if !ok {
		return fmt.Errorf("module.Storage does not support per-user append limit")
	}

	if cmd.Flags().Changed("value") {
		val, _ := cmd.Flags().GetInt("value")

		var err error
		if val == -1 {
			err = userAL.SetMessageLimit(nil)
		} else {
			val32 := uint32(val)
			err = userAL.SetMessageLimit(&val32)
		}
		if err != nil {
			return err
		}
	} else {
		lim := userAL.CreateMessageLimit()
		if lim == nil {
			fmt.Println("No limit")
		} else {
			fmt.Println(*lim)
		}
	}

	return nil
}
