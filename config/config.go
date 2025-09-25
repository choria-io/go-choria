// Copyright (c) 2018-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/choria-io/go-choria/build"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/confkey"
	"github.com/choria-io/go-choria/puppet"
)

// Config represents Choria cofnfiguration
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
	RegistrationSplay bool `confkey:"registration_splay" default:"true"`

	// The list of known Sub Collectives this node will join or communicate with, Servers will subscribe the node and each agent to each sub collective and Clients will publish to a chosen sub collective. Defaults to the build settin build.DefaultCollectives
	Collectives []string `confkey:"collectives" type:"comma_split"`

	// The Sub Collective where a Client will publish to when no specific Sub Collective is configured
	MainCollective string `confkey:"main_collective"`

	// The file to write logs to, when set to 'discard' logging will be disabled. Also supports 'stdout' and 'stderr' as special log destinations.
	LogFile string `confkey:"logfile" type:"path_string" default:"stdout"`

	// The lowest level log to add to the logfile
	LogLevel string `confkey:"loglevel" default:"info" validate:"enum=debug,info,warn,error,fatal"`

	// The directory where Agents, DDLs and other plugins are found
	LibDir []string `confkey:"libdir" type:"path_split"`

	// The identity this machine is known as, when empty it's derived based on the operating system hostname or by calling facter fqdn
	Identity string `confkey:"identity"`

	// Disables or enable CLI color
	Color bool `confkey:"color" default:"true"`

	// Path to a file listing configuration classes applied to a node, used in matches using Class filters
	ClassesFile string `confkey:"classesfile" default:"/opt/puppetlabs/puppet/cache/state/classes.txt" type:"path_string"`

	// How long to wait for responses while doing broadcast discovery
	DiscoveryTimeout int `confkey:"discovery_timeout" default:"2"`

	// When enabled uses rpcauditprovider to audit RPC requests processed by the server
	RPCAudit bool `confkey:"rpcaudit" default:"false" url:"https://choria.io/docs/configuration/aaa/"`

	// When enables authorization is performed on every RPC request based on rpcauthprovider
	RPCAuthorization bool `confkey:"rpcauthorization" default:"true" url:"https://choria.io/docs/configuration/aaa/"`

	// The Authorization system to use
	RPCAuthorizationProvider string `confkey:"rpcauthprovider" type:"title_string" default:"action_policy" url:"https://choria.io/docs/configuration/aaa/"`

	// When limiting nodes to a subset of discovered nodes this is the method to use, random is influenced by
	RPCLimitMethod string `confkey:"rpclimitmethod" default:"first" validate:"enum=first,random"`

	// How long published messages are allowed to linger on the network, lower numbers have a higher reliance on clocks being in sync
	TTL int `confkey:"ttl" default:"60"`

	// The default discovery plugin to use. The default "mc" uses a network broadcast, "choria" uses PuppetDB, external calls external commands
	DefaultDiscoveryMethod string `confkey:"default_discovery_method" default:"mc" validate:"enum=mc,broadcast,puppetdb,choria,external,inventory"`

	// Where to look for YAML or JSON based facts
	FactSourceFile string `confkey:"plugin.yaml" type:"path_string"`

	// Default options to pass to the discovery plugin
	DefaultDiscoveryOptions []string `confkey:"default_discovery_options"`

	// The amount of time to allow the server to exit, after this memory and thread dumps will be performed and a force exit will be done
	SoftShutdownTimeout int `confkey:"soft_shutdown_timeout" default:"2"`

	// ConfigFile is the main configuration that got parsed
	ConfigFile string

	// The system-wide configuration directory. Plugins are loaded from there
	SystemConfigDirectory string

	// ParsedFiles is a list of all files parsed to create the current config
	ParsedFiles []string

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

	// Allow things like completion to put the DDL Registry in cache-only mode
	RegistryCacheOnly bool

	// CustomLogger sets a logger instance that Choria framework will use and
	// not change any configuration, if you do this you should take care of
	// configuring the Logrus standard logger as some places Choria will log
	// via that
	CustomLogger *log.Logger
}

// NewDefaultSystemConfig creates a new configuration for system services
func NewDefaultSystemConfig(server bool) (*Config, error) {
	c := newConfig()
	c.InitiatedByServer = server

	err := c.normalize()
	if err != nil {
		return nil, err
	}

	return c, nil
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

func NewSystemConfig(path string, server bool) (*Config, error) {
	c := newConfig()
	c.InitiatedByServer = server

	err := loadConfigFiles(path, false, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func loadConfigFiles(path string, projects bool, c *Config) error {
	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(path)
	}

	c.ConfigFile = path

	err := parseConfig(path, c, "", c.rawOpts)
	if err != nil {
		return err
	}

	err = parseConfig(path, c.Choria, "", c.rawOpts)
	if err != nil {
		return err
	}

	err = c.parseAllDotCfg()
	if err != nil {
		return err
	}

	if projects {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		pfiles, err := ProjectConfigurationFiles(pwd)
		if err != nil {
			return err
		}

		for _, pp := range pfiles {
			err = parseConfig(pp, c, "", c.rawOpts)
			if err != nil {
				return err
			}

			err = parseConfig(pp, c.Choria, "", c.rawOpts)
			if err != nil {
				return err
			}
		}
	}

	return c.normalize()
}

// NewConfig parses a config file and return the config
func NewConfig(path string) (*Config, error) {
	c := newConfig()

	err := loadConfigFiles(path, true, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewConfigForTests creates a configuration for use in testing tools
func NewConfigForTests() *Config {
	c := newConfig()
	c.MainCollective = "ginkgo"
	c.Collectives = []string{"ginkgo", "mcollective"}
	c.RegistrationCollective = "ginkgo"
	c.Identity = "ginkgo.example.net"
	c.OverrideCertname = "rip.mcollective"
	c.LogLevel = "fatal"
	c.Choria.SSLDir = "/nonexisting"
	c.DisableSecurityProviderVerify = true
	c.LogFile = "discard"
	c.RPCAuthorization = false

	return c
}

func (c *Config) normalize() error {
	if len(c.Collectives) == 0 {
		c.Collectives = strings.Split(build.DefaultCollectives, ",")
		if len(c.Collectives) == 0 {
			c.Collectives = []string{"mcollective"}
		}

		for i, collective := range c.Collectives {
			c.Collectives[i] = strings.TrimSpace(collective)
		}

		// when using the choria security provider we switch default collectives
		if c.Choria.SecurityProvider == "choria" && len(c.Collectives) == 1 && c.Collectives[0] == "mcollective" {
			c.Collectives = []string{"choria"}
		}
	}

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
		if c.Choria.RemoteSignerURL == "" && !c.Choria.RemoteSignerService {
			return fmt.Errorf("anonymous TLS can only be enabled when a remote signer is configured")
		}

		c.DisableTLSVerify = true
		c.DisableSecurityProviderVerify = true
	}

	if c.Choria.ServerAnonTLS {
		c.DisableTLSVerify = true
		c.DisableSecurityProviderVerify = true

		if c.Choria.ServerTokenFile == "" {
			if c.ConfigFile == "" {
				return fmt.Errorf("cannot determine path to server token file")
			}
			c.Choria.ServerTokenFile = filepath.Join(filepath.Dir(c.ConfigFile), "server.jwt")
		}

		if c.Choria.ServerTokenSeedFile == "" {
			if c.ConfigFile == "" {
				return fmt.Errorf("cannot determine path to server token file")
			}
			c.Choria.ServerTokenSeedFile = filepath.Join(filepath.Dir(c.ConfigFile), "server.seed")
		}
	}

	if runtime.GOOS == "windows" {
		c.Color = false
	}

	if !c.Color {
		color.NoColor = true
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

// SetOption sets a raw string option, can be used to programmatically
// set plugin options etc, setting a main config item value here does
// not update the values in the strings, so this is only really useful
// for setting plugin options
func (c *Config) SetOption(option string, value string) {
	c.rawOpts[option] = value
}

// UnParsedOptions are the options loaded
func (c *Config) UnParsedOptions() map[string]string {
	return c.rawOpts
}

func (c *Config) dotdDir() string {
	return filepath.Join(c.SystemConfigDirectory, "plugin.d")
}

func newConfig() *Config {
	scd := ""
	if iu.FileExist("/etc/choria") {
		scd = "/etc/choria"
	} else if iu.FileExist("/usr/local/etc/choria") {
		scd = "/usr/local/etc/choria"
	} else if iu.FileExist("C:\\ProgramData\\choria\\etc") {
		scd = "C:\\ProgramData\\choria\\etc"
	}

	m := &Config{
		Choria:                newChoria(),
		rawOpts:               make(map[string]string),
		Puppet:                puppet.New(),
		SystemConfigDirectory: scd,
	}

	err := confkey.SetStructDefaults(m)
	if err != nil {
		log.Errorf("Config creation failed: %s", err)
	}

	return m
}
