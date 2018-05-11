package choria

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/choria-io/go-confkey"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/srvcache"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// ChoriaPluginConfig settings
type ChoriaPluginConfig struct {
	PuppetServerHost string `confkey:"plugin.choria.puppetserver_host" default:"puppet"`
	PuppetServerPort int    `confkey:"plugin.choria.puppetserver_port" default:"8140"`
	PuppetCAHost     string `confkey:"plugin.choria.puppetca_host" default:"puppet"`
	PuppetCAPort     int    `confkey:"plugin.choria.puppetca_port" default:"8140"`
	PuppetDBHost     string `confkey:"plugin.choria.puppetdb_host" default:"puppet"`
	PuppetDBPort     int    `confkey:"plugin.choria.puppetdb_port" default:"8081"`
	SSLDir           string `confkey:"plugin.choria.ssldir"`
	UseSRVRecords    bool   `confkey:"plugin.choria.use_srv" default:"true"`
	SRVDomain        string `confkey:"plugin.choria.srv_domain"`
	Provision        bool   `confkey:"plugin.choria.server.provision" default:"false"`

	// discovery proxy
	DiscoveryHost  string `confkey:"plugin.choria.discovery_host" default:"puppet"`
	DiscoveryPort  int    `confkey:"plugin.choria.discovery_port" default:"8085"`
	DiscoveryProxy bool   `confkey:"plugin.choria.discovery_proxy" default:"false"`

	// federation
	FederationCollectives     []string `confkey:"plugin.choria.federation.collectives" type:"comma_split" environment:"CHORIA_FED_COLLECTIVE"`
	FederationMiddlewareHosts []string `confkey:"plugin.choria.federation_middleware_hosts" type:"comma_split"`
	FederationCluster         string   `confkey:"plugin.choria.federation.cluster" default:"mcollective"`

	StatsListenAddress string `confkey:"plugin.choria.stats_address" default:"127.0.0.1"`
	StatsPort          int    `confkey:"plugin.choria.stats_port" default:"0"`

	// nats connector
	NatsUser                 string   `confkey:"plugin.nats.user" environment:"MCOLLECTIVE_NATS_USERNAME"`
	NatsPass                 string   `confkey:"plugin.nats.pass" environment:"MCOLLECTIVE_NATS_PASSWORD"`
	MiddlewareHosts          []string `confkey:"plugin.choria.middleware_hosts" type:"comma_split"`
	RandomizeMiddlewareHosts bool     `confkey:"plugin.choria.randomize_middleware_hosts" default:"false"`

	// network broker
	NetworkListenAddress string        `confkey:"plugin.choria.network.listen_address" default:"::"`
	NetworkClientPort    int           `confkey:"plugin.choria.network.client_port" default:"4222"`
	NetworkPeerPort      int           `confkey:"plugin.choria.network.peer_port" default:"5222"`
	NetworkPeerUser      string        `confkey:"plugin.choria.network.peer_user"`
	NetworkPeerPassword  string        `confkey:"plugin.choria.network.peer_password"`
	NetworkPeers         []string      `confkey:"plugin.choria.network.peers" type:"comma_split"`
	NetworkWriteDeadline time.Duration `confkey:"plugin.choria.network.write_deadline" type:"duration" default:"5s"`
	BrokerNetwork        bool          `confkey:"plugin.choria.broker_network" default:"false"`
	BrokerDiscovery      bool          `confkey:"plugin.choria.broker_discovery" default:"false"`
	BrokerFederation     bool          `confkey:"plugin.choria.broker_federation" default:"false"`

	// registration
	FileContentRegistrationData   string `confkey:"plugin.choria.registration.file_content.data" default:""`
	FileContentRegistrationTarget string `confkey:"plugin.choria.registration.file_content.target" default:""`
	FileContentCompression        bool   `confkey:"plugin.choria.registration.file_content.compression" default:"true"`

	// ruby compatibility
	RubyAgentShim   string   `confkey:"plugin.choria.agent_provider.mcorpc.agent_shim"`
	RubyAgentConfig string   `confkey:"plugin.choria.agent_provider.mcorpc.config"`
	RubyLibdir      []string `confkey:"plugin.choria.agent_provider.mcorpc.libdir" type:"path_split"`

	// security plugin
	SecurityProvider  string   `confkey:"plugin.security.provider" default:"puppet" validate:"enum=puppet,file"`
	PrivilegedUsers   []string `confkey:"plugin.choria.security.privileged_users" type:"comma_split" default:"\\.privileged.mcollective$"`
	CertnameWhitelist []string `confkey:"plugin.choria.security.certname_whitelist" type:"comma_split" default:"\\.mcollective$"`
	Serializer        string   `confkey:"plugin.choria.security.serializer" validate:"enum=json,yaml"`

	// file security
	FileSecurityCertificate string `confkey:"plugin.security.file.certificate"`
	FileSecurityKey         string `confkey:"plugin.security.file.key"`
	FileSecurityCA          string `confkey:"plugin.security.file.ca"`
	FileSecurityCache       string `confkey:"plugin.security.file.cache"`

	// adapters
	Adapters []string `confkey:"plugin.choria.adapters" type:"comma_split"`
}

// Config represents Choria configuration
type Config struct {
	Registration              []string `confkey:"registration" type:"comma_split" default:""`
	RegistrationCollective    string   `confkey:"registration_collective"`
	RegisterInterval          int      `confkey:"registerinterval" default:"300"`
	RegistrationSplay         bool     `confkey:"registration_splay" default:"false"`
	Collectives               []string `confkey:"collectives" type:"comma_split" default:"mcollective"`
	MainCollective            string   `confkey:"main_collective"`
	LogFile                   string   `confkey:"logfile"`
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
	ClassesFile               string   `confkey:"classesfile" default:"/opt/puppetlabs/puppet/cache/state/classes.txt"`
	DiscoveryTimeout          int      `confkey:"discovery_timeout" default:"2"`
	PublishTimeout            int      `confkey:"publish_timeout" default:"2"`
	ConnectionTimeout         int      `confkey:"connection_timeout"`
	RPCAudit                  bool     `confkey:"rpcaudit" default:"false"`
	RPCAuditProvider          string   `confkey:"rpcauditprovider" type:"title_string"`
	RPCAuthorization          bool     `confkey:"rpcauthorization" default:"false"`
	RPCAuthorizationProvider  string   `confkey:"rpcauthprovider" type:"title_string"`
	RPCLimitMethod            string   `confkey:"rpclimitmethod" default:"first"`
	LoggerType                string   `confkey:"logger_type" default:"file"`
	FactCacheTime             int      `confkey:"fact_cache_time" default:"300"`
	SSLCipher                 string   `confkey:"ssl_cipher" default:"aes-256-cbc"`
	Threaded                  bool     `confkey:"threaded" default:"false"`
	TTL                       int      `confkey:"ttl" default:"60"`
	DefaultDiscoveryOptions   []string `confkey:"default_discovery_options"`
	DefaultDiscoveryMethod    string   `confkey:"default_discovery_method" default:"mc"`
	SoftShutdown              bool     `confkey:"soft_shutdown" default:"false"`
	SoftShutdownTimeout       int      `confkey:"soft_shutdown_timeout"`
	ActivateAgents            bool     `confkey:"activate_agents" default:"true"`
	FactSourceFile            string   `confkey:"plugin.yaml" default:"/etc/puppetlabs/mcollective/generated-facts.yaml"`

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
}

// NewDefaultConfig creates a empty configuration
func NewDefaultConfig() (*Config, error) {
	c := newConfig()

	err := normalize(c)
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

	err = normalize(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func normalize(c *Config) error {
	var err error

	if c.MainCollective == "" {
		c.MainCollective = c.Collectives[0]
	}

	if c.RegistrationCollective == "" {
		c.RegistrationCollective = c.MainCollective
	}

	if c.Identity == "" {
		fqdn, _ := FacterFQDN()
		if fqdn != "" {
			c.Identity = fqdn
		} else {
			c.Identity, err = os.Hostname()
			if err != nil {
				return err
			}
		}
	}

	srvcache.SetIdentity(c.Identity)

	if build.TLS != "true" {
		c.DisableTLS = true
	}

	if c.LogLevel == "" {
		c.LogLevel = "debug"
	}

	if c.LogLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	}

	return nil
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
// can be used to extract the parsed settings
func parseDotConfFile(plugin string, conf *Config, target interface{}) error {
	cfgPath := filepath.Join(conf.dotdDir(), fmt.Sprintf("%s.cfg", plugin))
	if _, err := os.Stat(cfgPath); err == nil {
		err = parseConfig(cfgPath, target, fmt.Sprintf("plugin.%s", plugin), conf.rawOpts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (conf *Config) parseAllDotCfg() error {
	files, err := ioutil.ReadDir(conf.dotdDir())
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".cfg") {
			base := path.Base(file.Name())
			var target interface{}

			if base == "choria.cfg" {
				target = conf.Choria
			}

			plugin := strings.TrimSuffix(base, filepath.Ext(base))
			err := parseDotConfFile(plugin, conf, target)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (conf *Config) dotdDir() string {
	return filepath.Join(filepath.Dir(conf.ConfigFile), "plugin.d")
}

// parse a config file and fill in the given config structure based on its tags
func parseConfig(path string, config interface{}, prefix string, found map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	parseConfigContents(file, config, prefix, found)

	return nil
}

func parseConfigContents(content io.Reader, config interface{}, prefix string, found map[string]string) {
	scanner := bufio.NewScanner(content)
	itemr := regexp.MustCompile(`(.+?)\s*=\s*(.+)`)
	skipr := regexp.MustCompile(`^#|^$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !skipr.MatchString(line) {
			if itemr.MatchString(line) {
				matches := itemr.FindStringSubmatch(line)
				var key string

				if prefix == "" {
					key = matches[1]
				} else {
					key = prefix + "." + matches[1]
				}

				if config != nil {
					// errors here are normal since items for Choria and Config are in the same file
					confkey.SetStructFieldWithKey(config, key, matches[2])
				}

				found[key] = matches[2]
			}
		}
	}
}

func newConfig() *Config {
	m := &Config{
		Choria:  newChoria(),
		rawOpts: make(map[string]string),
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

func newChoria() *ChoriaPluginConfig {
	c := &ChoriaPluginConfig{}

	err := confkey.SetStructDefaults(c)
	if err != nil {
		log.Errorf("Choria config creation failed: %s", err)
	}

	return c
}
