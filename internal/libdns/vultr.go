//go:build libdns_vultr || !libdns_separate
// +build libdns_vultr !libdns_separate

package libdns

import (
	"github.com/libdns/vultr"
	"mailcoin/framework/config"
	"mailcoin/framework/module"
)

func init() {
	module.Register("libdns.vultr", func(modName, instName string, _, _ []string) (module.Module, error) {
		p := vultr.Provider{}
		return &ProviderModule{
			RecordDeleter:  &p,
			RecordAppender: &p,
			setConfig: func(c *config.Map) {
				c.String("api_token", false, false, "", &p.APIToken)
			},
			instName: instName,
			modName:  modName,
		}, nil
	})
}
