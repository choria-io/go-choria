package config

import (
	"github.com/choria-io/go-choria/confkey"
)

func (c *Config) DocForConfigKey(k string) *confkey.Doc {
	doc := confkey.KeyDoc(c, k, "")
	if doc != nil {
		return doc
	}

	return confkey.KeyDoc(c.Choria, k, "Choria")
}

// ConfigKeys retrieves all known configuration keys matching re
func (c *Config) ConfigKeys(re string) (found []string, err error) {
	found = []string{}

	keys, err := confkey.FindFields(c, re)
	if err != nil {
		return found, err
	}

	found = append(found, keys...)

	keys, err = confkey.FindFields(c.Choria, re)
	if err != nil {
		return found, err
	}

	found = append(found, keys...)

	return found, nil
}
