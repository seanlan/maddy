//go:build integration
// +build integration

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

package tests_test

import (
	"net"
	"strconv"
	"testing"

	"github.com/foxcpp/go-mockdns"
	"mailcoin/internal/testutils"
	"mailcoin/tests"
)

func TestMTA_Outbound(tt *testing.T) {
	t := tests.NewT(tt)
	t.DNS(map[string]mockdns.Zone{
		"example.invalid.": {
			MX: []net.MX{{Host: "mx.example.invalid.", Pref: 10}},
		},
		"mx.example.invalid.": {
			A: []string{"127.0.0.1"},
		},
	})
	t.Port("smtp")
	tgtPort := t.Port("remote_smtp")
	t.Config(`
		hostname mx.mailcoin.test
		tls off
		smtp tcp://127.0.0.1:{env:TEST_PORT_smtp} {
			deliver_to remote
		}`)
	t.Run(1)
	defer t.Close()

	be, s := testutils.SMTPServer(tt, "127.0.0.1:"+strconv.Itoa(int(tgtPort)))
	defer s.Close()

	c := t.Conn("smtp")
	defer c.Close()
	c.SMTPNegotation("client.mailcoin.test", nil, nil)
	c.Writeln("MAIL FROM:<from@mailcoin.test>")
	c.ExpectPattern("250 *")
	c.Writeln("RCPT TO:<to1@example.invalid>")
	c.ExpectPattern("250 *")
	c.Writeln("RCPT TO:<to2@example.invalid>")
	c.ExpectPattern("250 *")
	c.Writeln("DATA")
	c.ExpectPattern("354 *")
	c.Writeln("From: <from@mailcoin.test>")
	c.Writeln("To: <to@mailcoin.test>")
	c.Writeln("Subject: Hello!")
	c.Writeln("")
	c.Writeln("Hello!")
	c.Writeln(".")
	c.ExpectPattern("250 2.0.0 OK: queued")
	c.Writeln("QUIT")
	c.ExpectPattern("221 *")

	if be.SessionCounter != 1 {
		t.Fatal("No actual connection made?", be.SessionCounter)
	}
}

func TestIssue321(tt *testing.T) {
	t := tests.NewT(tt)
	t.DNS(map[string]mockdns.Zone{
		"example.invalid.": {
			AD: true,
			A:  []string{"127.0.0.1"},
		},
	})
	t.Port("smtp")
	tgtPort := t.Port("remote_smtp")
	t.Config(`
		hostname mx.mailcoin.test
		tls off
		smtp tcp://127.0.0.1:{env:TEST_PORT_smtp} {
			deliver_to remote {
				mx_auth {
					dnssec
					dane
				}
			}
		}`)
	t.Run(1)
	defer t.Close()

	be, s := testutils.SMTPServer(tt, "127.0.0.1:"+strconv.Itoa(int(tgtPort)))
	defer s.Close()

	c := t.Conn("smtp")
	defer c.Close()
	c.SMTPNegotation("client.mailcoin.test", nil, nil)
	c.Writeln("MAIL FROM:<from@mailcoin.test>")
	c.ExpectPattern("250 *")
	c.Writeln("RCPT TO:<to1@example.invalid>")
	c.ExpectPattern("250 *")
	c.Writeln("RCPT TO:<to2@example.invalid>")
	c.ExpectPattern("250 *")
	c.Writeln("DATA")
	c.ExpectPattern("354 *")
	c.Writeln("From: <from@mailcoin.test>")
	c.Writeln("To: <to@mailcoin.test>")
	c.Writeln("Subject: Hello!")
	c.Writeln("")
	c.Writeln("Hello!")
	c.Writeln(".")
	c.ExpectPattern("250 2.0.0 OK: queued")
	c.Writeln("QUIT")
	c.ExpectPattern("221 *")

	if be.SessionCounter != 1 {
		t.Fatal("No actual connection made?", be.SessionCounter)
	}
}
