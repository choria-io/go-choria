package choria

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

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

	// security plugin
	PrivilegedUsers   []string `confkey:"plugin.choria.security.privileged_users" type:"comma_split" default:"\\.privileged.mcollective$"`
	CertnameWhitelist []string `confkey:"plugin.choria.security.certname_whitelist" type:"comma_split" default:"\\.mcollective$"`
	Serializer        string   `confkey:"plugin.choria.security.serializer"` // TODO support enums

	// network broker
	NetworkListenAddress string   `confkey:"plugin.choria.network.listen_address" default:"::"`
	NetworkClientPort    int      `confkey:"plugin.choria.network.client_port" default:"4222"`
	NetworkPeerPort      int      `confkey:"plugin.choria.network.peer_port" default:"5222"`
	NetworkPeerUser      string   `confkey:"plugin.choria.network.peer_user"`
	NetworkPeerPassword  string   `confkey:"plugin.choria.network.peer_password"`
	NetworkPeers         []string `confkey:"plugin.choria.network.peers" type:"comma_split"`
	BrokerNetwork        bool     `confkey:"plugin.choria.broker_network" default:"false"`
	BrokerDiscovery      bool     `confkey:"plugin.choria.broker_discovery" default:"false"`
	BrokerFederation     bool     `confkey:"plugin.choria.broker_federation" default:"false"`

	// discovery
	FactSourceFile string `confkey:"plugin.yaml" default:"/etc/puppetlabs/mcollective/generated-facts.yaml"`

	// registration
	FileContentRegistrationData   string `confkey:"plugin.choria.registration.file_content.data" default:""`
	FileContentRegistrationTarget string `confkey:"plugin.choria.registration.file_content.target" default:""`

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
	LogLevel                  string   `confkey:"loglevel" default:"info"` // TODO support enums
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
	RPCLimitMethod            string   `confkey:"rpclimitmethod" default:"first"` // TODO support enums
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

	ConfigFile string

	// the options exactly as they were found in the config files
	rawOpts map[string]string

	Choria *ChoriaPluginConfig

	// options that are not user configurable via config files but can be
	// used by things like the emulator to set up a TLS free setup
	DisableTLS       bool
	DisableTLSVerify bool
	OverrideCertname string
}

// HasOption determines if a specific option was set from a config key.
// The option given would be something like `plugin.choria.use_srv`
// and true would indicate that it was set by config vs using defaults
func (self *Config) HasOption(option string) bool {
	_, ok := self.rawOpts[option]

	return ok
}

// Option retrieves the raw string representation of a given option
// from that was loaded from the configuration
func (self *Config) Option(option string, deflt string) string {
	v, ok := self.rawOpts[option]

	if !ok {
		return deflt
	}

	return v
}

// NewConfig parses a config file and return the config
func NewConfig(path string) (*Config, error) {
	c := newConfig()
	c.ConfigFile = path
	c.rawOpts = make(map[string]string)

	// TODO i think probably parse config can walk 'mcollective' recursively
	err := parseConfig(path, c, "", c.rawOpts)
	if err != nil {
		return nil, err
	}

	err = parseConfig(path, c.Choria, "", c.rawOpts)
	if err != nil {
		return nil, err
	}

	choriaPConf := filepath.Join(filepath.Dir(path), "plugin.d", "choria.cfg")
	if _, err := os.Stat(choriaPConf); err == nil {
		err = parseConfig(choriaPConf, c.Choria, "plugin.choria", c.rawOpts)
		if err != nil {
			return nil, err
		}
	}

	if c.MainCollective == "" {
		c.MainCollective = c.Collectives[0]
	}

	if c.RegistrationCollective == "" {
		c.RegistrationCollective = c.MainCollective
	}

	if c.Identity == "" {
		c.Identity, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	srvcache.SetIdentity(c.Identity)

	if build.TLS != "true" {
		c.DisableTLS = true
	}

	// TODO other loglevels, not needed for this project
	if c.LogLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	}

	return c, nil
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

				setItemWithKey(config, key, matches[2])
				found[key] = matches[2]
			}
		}
	}
}

func newConfig() *Config {
	m := &Config{Choria: newChoria()}
	setDefaults(m)

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		m.Color = false
	}

	return m
}

func newChoria() *ChoriaPluginConfig {
	c := &ChoriaPluginConfig{}
	setDefaults(c)

	return c
}

// finds the struct key that matches the confkey on s and assign the value to it
func setItemWithKey(s interface{}, key string, value interface{}) error {
	item, err := itemWithKey(s, key)
	if err != nil {
		return err
	}

	if t, ok := tag(s, item, "environment"); ok {
		if v, ok := os.LookupEnv(t); ok {
			value = v
		}
	}

	field := reflect.ValueOf(s).Elem().FieldByName(item)

	switch field.Kind() {
	case reflect.Slice:
		ptr := field.Addr().Interface().(*[]string)

		if t, ok := tag(s, item, "type"); ok {
			switch t {
			case "comma_split":
				// specifically clear it since these are one line split like 'collectives'
				*ptr = []string{}
				vals := strings.Split(value.(string), ",")

				for _, v := range vals {
					*ptr = append(*ptr, strings.TrimSpace(v))
				}

			case "path_split":
				// these are like libdir, either a one line split or a multiple occurance with splits
				vals := strings.Split(value.(string), string(os.PathListSeparator))

				for _, v := range vals {
					*ptr = append(*ptr, strings.TrimSpace(v))
				}
			}
		} else {
			*ptr = append(*ptr, strings.TrimSpace(value.(string)))
		}

	case reflect.Int:
		ptr := field.Addr().Interface().(*int)
		i, err := strconv.Atoi(value.(string))
		if err != nil {
			return err
		}
		*ptr = i

	case reflect.String:
		ptr := field.Addr().Interface().(*string)
		*ptr = value.(string)

		if t, ok := tag(s, item, "type"); ok {
			if t == "title_string" {
				a := []rune(value.(string))
				a[0] = unicode.ToUpper(a[0])
				*ptr = string(a)
			}
		}

	case reflect.Bool:
		ptr := field.Addr().Interface().(*bool)
		b, _ := StrToBool(value.(string))
		*ptr = b
	}

	return nil
}

// determines the struct key name that is tagged with a certain confkey
func itemWithKey(s interface{}, key string) (string, error) {
	st := reflect.TypeOf(s)
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	for i := 0; i <= st.NumField()-1; i++ {
		field := st.Field(i)

		if confkey, ok := field.Tag.Lookup("confkey"); ok {
			if confkey == key {
				return field.Name, nil
			}
		}
	}

	return "", errors.New("Can't find any structure element that holds " + key)
}

// extract defaults out of the tags and set them to the key
func setDefaults(s interface{}) {
	st := reflect.TypeOf(s).Elem()

	for i := 0; i <= st.NumField()-1; i++ {
		field := st.Field(i)

		if key, ok := field.Tag.Lookup("confkey"); ok {
			if value, ok := field.Tag.Lookup("default"); ok {
				setItemWithKey(s, key, value)
			}
		}
	}
}

// retrieve a tag for a struct field
func tag(s interface{}, field string, tag string) (string, bool) {
	st := reflect.TypeOf(s)

	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	for i := 0; i <= st.NumField()-1; i++ {
		f := st.Field(i)

		if f.Name == field {
			if value, ok := f.Tag.Lookup(tag); ok {
				return value, true
			}
		}
	}

	return "", false
}
