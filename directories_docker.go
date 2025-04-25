//go:build docker
// +build docker

package mailcoin

var (
	ConfigDirectory         = "/data"
	DefaultStateDirectory   = "/data"
	DefaultRuntimeDirectory = "/tmp"
	DefaultLibexecDirectory = "/usr/lib/mailcoin"
)
