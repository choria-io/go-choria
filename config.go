package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choria-io/go-confkey"

	puppet "github.com/choria-io/go-puppet"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// Config represents Choria configuration
type Config struct {
	Registration              []string `confkey:"registration" type:"comma_split" default:""`
	RegistrationCollective    string   `confkey:"registration_collective"`
	RegisterInterval          int      `confkey:"registerinterval" default:"300"`
	RegistrationSplay         bool     `confkey:"registration_splay" default:"false"`
	Collectives               []string `confkey:"collectives" type:"comma_split" default:"mcollective"`
	MainCollective            string   `confkey:"main_collective"`
	LogFile                   string   `confkey:"logfile" type:"path_string"`
	KeepLogs                  int      `confkey:"keeplogs" default:"5"`
	MaxLogSize                int      `confkey:"max_log_size" default:"2097152"`
	LogLevel                  string   `confkey:"loglevel" default:"info" validate:"enum=debug,info,warn,error,fatal"`
	LogFacility               string   `confkey:"logfacility" default:"user"`
	LibDir                    []string `confkey:"libdir" type:"path_split"`
	Identity                  string   `confkey:"identity"`
	DirectAddressing          bool     `confkey:"direct_addressing" default:"true"`
	DirectAddressingThreshold int      `confkey:"direct_addressing_threshold" default:"10"`
	Color                     bool     `confkey:"color" default:"true"`
	Daemonize                 bool     `confkey:"daemonize" default:"false"`
	SecurityProvider          string   `confkey:"securityprovider" default:"psk" type:"title_string"`
	FactSource                string   `confkey:"factsource" default:"yaml" default:"yaml"`
	Connector                 string   `confkey:"connector" default:"nats" type:"title_string"`
	ClassesFile               string   `confkey:"classesfile" default:"/opt/puppetlabs/puppet/cache/state/classes.txt" type:"path_string"`
	DiscoveryTimeout          int      `confkey:"discovery_timeout" default:"2"`
	PublishTimeout            int      `confkey:"publish_timeout" default:"2"`
	ConnectionTimeout         int      `confkey:"connection_timeout"`
	RPCAudit                  bool     `confkey:"rpcaudit" default:"false"`
	RPCAuditProvider          string   `confkey:"rpcauditprovider" type:"title_string"`
	RPCAuthorization          bool     `confkey:"rpcauthorization" default:"false"`
	RPCAuthorizationProvider  string   `confkey:"rpcauthprovider" type:"title_string" default:"action_policy"`
	RPCLimitMethod            string   `confkey:"rpclimitmethod" default:"first" validate:"enum=first,random"`
	LoggerType                string   `confkey:"logger_type" default:"file"`
	FactCacheTime             int      `confkey:"fact_cache_time" default:"300"`
	SSLCipher                 string   `confkey:"ssl_cipher" default:"aes-256-cbc"`
	Threaded                  bool     `confkey:"threaded" default:"false"`
	TTL                       int      `confkey:"ttl" default:"60"`
	DefaultDiscoveryOptions   []string `confkey:"default_discovery_options"`
	DefaultDiscoveryMethod    string   `confkey:"default_discovery_method" default:"mc"`
	SoftShutdown              bool     `confkey:"soft_shutdown" default:"true"`
	SoftShutdownTimeout       int      `confkey:"soft_shutdown_timeout" default:"2"`
	ActivateAgents            bool     `confkey:"activate_agents" default:"true"`
	FactSourceFile            string   `confkey:"plugin.yaml" default:"/etc/puppetlabs/mcollective/generated-facts.yaml" type:"path_string"`

	ConfigFile string

	// the options exactly as they were found in the config files
	rawOpts map[string]string

	Choria *ChoriaPluginConfig

	// options that are not user configurable via config files but can be
	// used by things like the emulator to set up a TLS free setup

	// DisableSecurityProviderVerify skips calling security provider Validate()
	DisableSecurityProviderVerify bool

	// DisableTLS turns off TLS and skips calling security provider Validate()
	DisableTLS bool

	// DisableTLSVerify turns off CA validation etc in TLS connections
	DisableTLSVerify bool

	// OverrideCertname sets a arbitrary certname and short circuits calling Puppet etc
	// this is mainly used by tests to adjust the certname on the fly
	OverrideCertname string

	// InitiatedByServer indicates to the framework that certain server specific
	// initialization steps - like Provisioning mode - should be performed.
	InitiatedByServer bool

	// Puppet provides access to puppet config data, settings and facts
	Puppet *puppet.PuppetWrapper
}

// NewDefaultConfig creates a empty configuration
func NewDefaultConfig() (*Config, error) {
	c := newConfig()

	err := c.normalize()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewConfig parses a config file and return the config
func NewConfig(path string) (*Config, error) {
	c := newConfig()
	c.ConfigFile = path

	err := parseConfig(path, c, "", c.rawOpts)
	if err != nil {
		return nil, err
	}

	err = parseConfig(c.ConfigFile, c.Choria, "", c.rawOpts)
	if err != nil {
		return nil, err
	}

	c.parseAllDotCfg()

	err = c.normalize()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func NewConfigForTests() *Config {
	c := newConfig()
	c.MainCollective = "ginkgo"
	c.RegistrationCollective = "ginkgo"
	c.Identity = "ginkgo.example.net"
	c.OverrideCertname = "rip.mcollective"
	c.LogLevel = "fatal"

	return c
}

func (c *Config) normalize() error {
	if c.MainCollective == "" {
		c.MainCollective = c.Collectives[0]
	}

	if c.RegistrationCollective == "" {
		c.RegistrationCollective = c.MainCollective
	}

	if c.Identity == "" {
		hn, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("could not determine hostname: %s", err)
		}

		// if os.Hostname gets a full hostname use that as it's quicker, then try facter if
		// that's not available then use whatever os.Hostname gave even if its a short name
		if strings.Count(hn, ".") > 1 {
			c.Identity = hn
		} else if fqdn, _ := DNSFQDN(); fqdn != "" {
			c.Identity = fqdn
		} else if fqdn, _ := c.Puppet.FacterFQDN(); fqdn != "" {
			c.Identity = fqdn
		} else {
			c.Identity = hn
		}

		if c.Identity == "" {
			return errors.New("could not determine identity from os.Hostname or facter, please set identity in the configuration")
		}
	}

	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}

	if c.LogLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	}

	return nil
}

// BuildInfoProvider provides build time information
type BuildInfoProvider interface {
	HasTLS() bool
}

// ApplyBuildSettings applies build time overrides to the configuration
func (conf *Config) ApplyBuildSettings(b BuildInfoProvider) {
	conf.DisableTLS = !b.HasTLS()
}

// HasOption determines if a specific option was set from a config key.
// The option given would be something like `plugin.choria.use_srv`
// and true would indicate that it was set by config vs using defaults
func (conf *Config) HasOption(option string) bool {
	_, ok := conf.rawOpts[option]

	return ok
}

// Option retrieves the raw string representation of a given option
// from that was loaded from the configuration
func (conf *Config) Option(option string, deflt string) string {
	v, ok := conf.rawOpts[option]

	if !ok {
		return deflt
	}

	return v
}

// parseDotConfFile parses a file like /etc/..../plugin.d/package.cfg as if its full of
// plugin.package.x = y lines and fill in a structure with the results if that structure
// declares its options using the same tag structure as Config.
//
// If the supplied target structure is nil then the only side effect will be that the
// supplied conf will be updated with the raw options so that HasOption() and Option()
func (conf *Config) dotdDir() string {
	return filepath.Join(filepath.Dir(conf.ConfigFile), "plugin.d")
}

func newConfig() *Config {
	m := &Config{
		Choria:  newChoria(),
		rawOpts: make(map[string]string),
		Puppet:  puppet.New(),
	}

	err := confkey.SetStructDefaults(m)
	if err != nil {
		log.Errorf("Config creation failed: %s", err)
	}

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		m.Color = false
	}

	return m
}
