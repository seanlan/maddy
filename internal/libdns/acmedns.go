//go:build libdns_acmedns || libdns_all
// +build libdns_acmedns libdns_all

package libdns

import (
	"github.com/libdns/acmedns"
	"mailcoin/framework/config"
	"mailcoin/framework/module"
)

func init() {
	module.Register("libdns.acmedns", func(modName, instName string, _, _ []string) (module.Module, error) {
		p := acmedns.Provider{}
		return &ProviderModule{
			RecordDeleter:  &p,
			RecordAppender: &p,
			setConfig: func(c *config.Map) {
				c.String("username", false, true, "", &p.Username)
				c.String("password", false, true, "", &p.Password)
				c.String("subdomain", false, true, "", &p.Subdomain)
				c.String("server_url", false, true, "", &p.ServerURL)
			},
			instName: instName,
			modName:  modName,
		}, nil
	})
}
