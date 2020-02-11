package config

import (
	"sort"

	"github.com/choria-io/go-choria/confkey"
)

func (c *Config) overrideDocDescription(doc *confkey.Doc) *confkey.Doc {
	if doc == nil {
		return doc
	}

	desc, ok := docStrings[doc.ConfigKey()]
	if ok {
		doc.SetDescription(desc)
	}

	return doc
}

func (c *Config) DocForConfigKey(k string) *confkey.Doc {
	doc := confkey.KeyDoc(c, k, "")
	if doc != nil {
		return c.overrideDocDescription(doc)
	}

	doc = confkey.KeyDoc(c.Choria, k, "Choria")

	return c.overrideDocDescription(doc)
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

	sort.Strings(found)

	return found, nil
}
