package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/choria-io/go-choria/confkey"
	"github.com/choria-io/go-choria/puppet"
)

// Config represents Choria configuration
//
// NOTE: When adding or updating doc strings please run `go generate` in the root of the repository
type Config struct {
	// The plugins used when publishing Registration data, when this is unset or empty sending registration data is disabled
	Registration []string `confkey:"registration" type:"comma_split"`

	// The Sub Collective to publish registration data to
	RegistrationCollective string `confkey:"registration_collective"`

	// How often to publish registration data
	RegisterInterval int `confkey:"registerinterval" default:"300"`

	// When true delays initial registration publish by a random period up to registerinterval following registration publishes will be at registerinterval without further splay
	RegistrationSplay bool `confkey:"registration_splay" default:"false"`

	// The list of known Sub Collectives this node will join or communicate with, Servers will subscribe the node and each agent to each sub collective and Clients will publish to a chosen sub collective
	Collectives []string `confkey:"collectives" type:"comma_split" default:"mcollective"`

	// The Sub Collective where a Client will publish to when no specific Sub Collective is configured
	MainCollective string `confkey:"main_collective"`

	// The file to write logs to, when set to an empty string logging will be to the console
	LogFile string `confkey:"logfile" type:"path_string"`

	// The lowest level log to add to the logfile
	LogLevel string `confkey:"loglevel" default:"info" validate:"enum=debug,info,warn,error,fatal"`

	// The directory where Agents, DDLs and other plugins are found
	LibDir []string `confkey:"libdir" type:"path_split"`

	// The identity this machine is known as, when empty it's derived based on the operating system hostname or by calling facter fqdn
	Identity string `confkey:"identity"`

	// Enables the direct-to-node communications pattern, unused in the Go clients
	DirectAddressing bool `confkey:"direct_addressing" default:"true"`

	// Disables or enable CLI color, not well supported in Go based code
	Color bool `confkey:"color" default:"true"`

	// Used to select the security provider in Ruby clients, only sensible value is "choria"
	SecurityProvider string `confkey:"securityprovider" default:"choria" type:"title_string" deprecated:"1"`

	// Configures the network connector to use, only sensible value is "nats", unused in Go based code
	Connector string `confkey:"connector" default:"nats" type:"title_string"`

	// Path to a file listing configuration classes applied to a node, used in matches using Class filters
	ClassesFile string `confkey:"classesfile" default:"/opt/puppetlabs/puppet/cache/state/classes.txt" type:"path_string"`

	// How long to wait for responses while doing broadcast discovery
	DiscoveryTimeout int `confkey:"discovery_timeout" default:"2"`

	// Ruby clients use this to determine how long they will allow when publishing requests
	PublishTimeout int `confkey:"publish_timeout" default:"2"`

	// Ruby clients use this to determine how long they will try to connect, fails after timeout
	ConnectionTimeout int `confkey:"connection_timeout"`

	// When enabled uses rpcauditprovider to audit RPC requests processed by the server
	RPCAudit bool `confkey:"rpcaudit" default:"false" url:"https://choria.io/docs/configuration/aaa/"`

	// The audit provider to use, unused at present as there is only a "choria" one
	RPCAuditProvider string `confkey:"rpcauditprovider" type:"title_string" url:"https://choria.io/docs/configuration/aaa/"`

	// When enables authorization is performed on every RPC request based on rpcauthprovider
	RPCAuthorization bool `confkey:"rpcauthorization" default:"false" url:"https://choria.io/docs/configuration/aaa/"`

	// The Authorization system to use
	RPCAuthorizationProvider string `confkey:"rpcauthprovider" type:"title_string" default:"action_policy" url:"https://choria.io/docs/configuration/aaa/"`

	// When limiting nodes to a subset of discovered nodes this is the method to use, random is influenced by
	RPCLimitMethod string `confkey:"rpclimitmethod" default:"first" validate:"enum=first,random"`

	// The type of logging to use, unused in Go based programs
	LoggerType string `confkey:"logger_type" default:"file" validate:"enum=console,file,syslog"`

	// Enables multi threaded mode in the Ruby client, generally a bad idea
	Threaded bool `confkey:"threaded" default:"false"`

	// How long published messages are allowed to linger on the network, lower numbers have a higher reliance on clocks being in sync
	TTL int `confkey:"ttl" default:"60"`

	// Configurable options to always pass to the discovery subsystem
	DefaultDiscoveryOptions []string `confkey:"default_discovery_options"`

	// The default discovery plugin to use. The default "mc" uses a network broadcast and "choria" uses PuppetDB
	DefaultDiscoveryMethod string `confkey:"default_discovery_method" default:"mc"`

	// Where to look for YAML or JSON based facts
	FactSourceFile string `confkey:"plugin.yaml" default:"/etc/puppetlabs/mcollective/generated-facts.yaml" type:"path_string"`

	// Deprecated settings

	ActivateAgents            bool   `confkey:"activate_agents" default:"true" deprecated:"1"`
	Daemonize                 bool   `confkey:"daemonize" default:"false" deprecated:"1"`
	DirectAddressingThreshold int    `confkey:"direct_addressing_threshold" default:"10" deprecated:"1"`
	FactCacheTime             int    `confkey:"fact_cache_time" default:"300" deprecated:"1"`
	FactSource                string `confkey:"factsource" default:"yaml" deprecated:"1"`
	KeepLogs                  int    `confkey:"keeplogs" default:"5" deprecated:"1"`
	LogFacility               string `confkey:"logfacility" default:"user" deprecated:"1"`
	MaxLogSize                int    `confkey:"max_log_size" default:"2097152" deprecated:"1"`
	SoftShutdown              bool   `confkey:"soft_shutdown" default:"true" deprecated:"1"`
	SoftShutdownTimeout       int    `confkey:"soft_shutdown_timeout" default:"2" deprecated:"1"`

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
	Puppet *puppet.Wrapper

	// CacheBatchedTransports should be true when a agent provider does batched
	// requests where effectively the same request can span many publishes often
	// long apart. The problem is that in these cases the security framework might
	// require frequent 2FA and users might be prompted for 2FA mid-batch.  This
	// setting will hint to choria.Message to return the same transport message
	// repeatedly
	CacheBatchedTransports bool
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

// NewConfigForTests creates a configuration for use in testing tools
func NewConfigForTests() *Config {
	c := newConfig()
	c.MainCollective = "ginkgo"
	c.RegistrationCollective = "ginkgo"
	c.Identity = "ginkgo.example.net"
	c.OverrideCertname = "rip.mcollective"
	c.LogLevel = "fatal"
	c.Choria.SSLDir = "/nonexisting"

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
		//
		// kubernetes does not have domain names in the pod hosts so we just take whats there
		// when running in a pod
		if strings.Count(hn, ".") > 1 {
			c.Identity = hn
		} else if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			c.Identity = hn
			fqdn, err := DNSFQDN()
			if err == nil {
				c.Identity = fqdn
			}
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

	if c.Choria.ClientAnonTLS {
		if c.Choria.RemoteSignerURL == "" && c.Choria.RemoteSignerSigningCert == "" {
			return fmt.Errorf("anonymous TLS can only be enabled when a remote signer is configured")
		}

		c.DisableTLSVerify = true
		c.DisableSecurityProviderVerify = true
	}

	return nil
}

// BuildInfoProvider provides build time information
type BuildInfoProvider interface {
	HasTLS() bool
}

// ApplyBuildSettings applies build time overrides to the configuration
func (c *Config) ApplyBuildSettings(b BuildInfoProvider) {
	c.DisableTLS = !b.HasTLS()
}

// HasOption determines if a specific option was set from a config key.
// The option given would be something like `plugin.choria.use_srv`
// and true would indicate that it was set by config vs using defaults
func (c *Config) HasOption(option string) bool {
	_, ok := c.rawOpts[option]

	return ok
}

// Option retrieves the raw string representation of a given option
// from that was loaded from the configuration
func (c *Config) Option(option string, deflt string) string {
	v, ok := c.rawOpts[option]

	if !ok {
		return deflt
	}

	return v
}

// SetOption sets a raw string option, can be used to programatically
// set plugin options etc, setting a main config item value here does
// not update the values in the strings, so this is only really useful
// for setting plugin options
func (c *Config) SetOption(option string, value string) {
	c.rawOpts[option] = value
}

// parseDotConfFile parses a file like /etc/..../plugin.d/package.cfg as if its full of
// plugin.package.x = y lines and fill in a structure with the results if that structure
// declares its options using the same tag structure as Config.
//
// If the supplied target structure is nil then the only side effect will be that the
// supplied conf will be updated with the raw options so that HasOption() and Option()
func (c *Config) dotdDir() string {
	return filepath.Join(filepath.Dir(c.ConfigFile), "plugin.d")
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
