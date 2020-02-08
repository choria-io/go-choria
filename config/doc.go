package config

import (
	"github.com/choria-io/go-choria/confkey"
)

var docSrtings = map[string]Doc{
	"registration":                    Doc{ds: "The plugin to use for sending Registration data"},
	"registration_collective":         Doc{ds: "The Sub Collective to publish registration data to"},
	"registerinterval":                Doc{ds: "How often to publish registration data"},
	"registration_splay":              Doc{ds: "When true delays initial registration publish by a random period up to registerinterval"},
	"collectives":                     Doc{ds: "The list of known Sub Collectives this node will join or communicate with"},
	"main_collective":                 Doc{ds: "The collective to publish to when no specific Sub Collective is configured"},
	"logfile":                         Doc{ds: "The file to write logs to"},
	"loglevel":                        Doc{ds: "The lowest level log to add to the logfile"},
	"max_log_size":                    Doc{ds: "Maximum size a log file will be before being rotated", deprecated: true},
	"keeplogs":                        Doc{ds: "How many rotated log files to keep", deprecated: true},
	"logfacility":                     Doc{ds: "When logging to syslog what facility to log with", deprecated: true},
	"plugin.choria.puppetserver_host": Doc{ds: "The hostname where your Puppet Server can be found"},
	"plugin.choria.puppetserver_port": Doc{ds: "The port your Puppet Server listens on"},
	"plugin.choria.puppetca_host":     Doc{ds: "The hostname where your Puppet Certificate Authority can be found"},
	"plugin.choria.puppetca_port":     Doc{ds: "The port your Puppet Certificate Authority listens on"},
}

type Doc struct {
	ds         string
	url        string
	deprecated bool
	structKey  string
	configKey  string
	container  string
	dflt       string
	env        string
	vtype      string
	validation string
}

// Deprecated indicates if the item is not in use anymore
func (d *Doc) Deprecate() bool {
	return d.deprecated
}

// StructKey is the key within the structure to lookup to retrieve the item
func (d *Doc) StructKey() string {
	if d.container != "" {
		return d.container + "." + d.structKey
	}

	return d.structKey
}

// ConfigKey is the key to place within the configuration to set the item
func (d *Doc) ConfigKey() string {
	return d.configKey
}

// Type is the type of data to store in the item
func (d *Doc) Type() string {
	return d.vtype
}

// Description is a description of the item, empty when not set
func (d *Doc) Description() string {
	if d.ds == "" {
		return "Undocumented"
	}

	return d.ds
}

// URL returns a url that describes the related feature in more detail
func (d *Doc) URL() string {
	return d.url
}

// Default is the default value as a string
func (d *Doc) Default() string {
	return d.dflt
}

// Validation is the configured validation
func (d *Doc) Validation() string {
	return d.validation
}

// Environment is an environment variable that can set this item, empty when not settable
func (d *Doc) Environment() string {
	return d.env
}

func (c *Config) DocForConfigKey(k string) *Doc {
	var err error

	d, ok := docSrtings[k]
	if !ok {
		d = Doc{}
	}

	d.configKey = k
	d.structKey, err = confkey.FieldWithKey(c, k)
	if err != nil {
		d.structKey, err = confkey.FieldWithKey(c.Choria, k)
		if err != nil {
			return nil
		}

		d.container = "Choria"
		d.dflt, _ = confkey.DefaultString(c.Choria, k)
		d.env, _ = confkey.Environment(c.Choria, k)
		d.vtype, _ = confkey.Type(c.Choria, k)
		d.validation, _ = confkey.Validation(c.Choria, k)

		return &d
	}

	d.dflt, _ = confkey.DefaultString(c, k)
	d.env, _ = confkey.Environment(c, k)
	d.vtype, _ = confkey.Type(c, k)
	d.validation, _ = confkey.Validation(c, k)

	return &d
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
