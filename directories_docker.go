//go:build docker
// +build docker

package mailchat

var (
	ConfigDirectory         = "/data"
	DefaultStateDirectory   = "/data"
	DefaultRuntimeDirectory = "/tmp"
	DefaultLibexecDirectory = "/usr/lib/mailchat"
)
