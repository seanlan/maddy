//go:build libdns_gcore || !libdns_separate
// +build libdns_gcore !libdns_separate

package libdns

import (
	"fmt"

	"github.com/libdns/gcore"
	"mailcoin/framework/config"
	"mailcoin/framework/module"
)

func init() {
	module.Register("libdns.gcore", func(modName, instName string, _, _ []string) (module.Module, error) {
		p := gcore.Provider{}
		return &ProviderModule{
			RecordDeleter:  &p,
			RecordAppender: &p,
			setConfig: func(c *config.Map) {
				c.String("api_key", false, false, "", &p.APIKey)
			},
			afterConfig: func() error {
				if p.APIKey == "" {
					return fmt.Errorf("libdns.gcore: api_key should be specified")
				}
				return nil
			},

			instName: instName,
			modName:  modName,
		}, nil
	})
}
