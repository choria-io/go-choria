package mcollective

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

	// discovery proxy
	DiscoveryHost  string `confkey:"plugin.choria.discovery_host" default:"puppet"`
	DiscoveryPort  int    `confkey:"plugin.choria.discovery_port" default:"8085"`
	DiscoveryProxy bool   `confkey:"plugin.choria.discovery_proxy" default:"false"`

	// federation
	FederationCollectives []string `confkey:"plugin.choria.federation.collectives" type:"comma_split" environment:"CHORIA_FED_COLLECTIVE"`
	StatsPort             int      `configkey:"plugin.choria.stats_port"`

	// nats connector
	NatsUser                  string   `confkey:"plugin.nats.user" environment:"MCOLLECTIVE_NATS_USERNAME"`
	NatsPass                  string   `confkey:"plugin.nats.pass" environment:"MCOLLECTIVE_NATS_PASSWORD"`
	MiddlewareHosts           []string `confkey:"plugin.choria.middleware_hosts" type:"comma_split"`
	FederationMiddlewareHosts []string `confkey:"plugin.choria.federation_middleware_hosts" type:"comma_split"`
	RandomizeMiddlewareHosts  bool     `confkey:"plugin.choria.randomize_middleware_hosts" default:"false"`

	// security plugin
	PrivilegedUsers   []string `confkey:"plugin.choria.security.privileged_users" type:"comma_split"`
	CertnameWhitelist []string `confkey:"plugin.choria.security.certname_whitelist" type:"comma_split"`
	Serializer        string   `confkey:"plugin.choria.security.serializer"` // TODO support enums

	// network broker
	NetworkClientPort   int      `confkey:"plugin.choria.network_client_port" default:"4222"`
	NetworkPeerPort     int      `confkey:"plugin.choria.network_peer_port" default:"5222"`
	NetworkPeerUser     string   `confkey:"plugin.choria.network_peer_user"`
	NetworkPeerPassword string   `confkey:"plugin.choria.network_peer_password"`
	NetworkPeers        []string `confkey:"plugin.choria.network_peers" type:"comma_split"`
	BrokerNetwork       bool     `confkey:"plugin.choria.broker_network" default:"false"`
	BrokerDiscovery     bool     `confkey:"plugin.choria.broker_discovery" default:"false"`
	BrokerFederation    bool     `confkey:"plugin.choria.broker_federation" default:"false"`
	FederationCluster   string   `confkey:"plugin.choria.broker_federation_cluster" default:"mcollective"`
}

// MCOConfig represents MCollective configuration
type MCOConfig struct {
	Registration              string   `confkey:"registration" default:"Agentlist" type:"title_string"`
	RegistrationCollective    string   `confkey:"registration_collective"`
	RegisterInterval          int      `confkey:"registerinterval" default:"0"`
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
	FactSource                string   `confkey:"factsource" default:"yaml" type:"title_string"`
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

	// list of all the options that were actually set
	setOptions []string

	Choria *ChoriaPluginConfig
}

// HasOption determines if a specific option was set from a config key.
// The option given would be something like `plugin.choria.use_srv`
// and true would indicate that it was set by config vs using defaults
func (c *MCOConfig) HasOption(option string) bool {
	for _, i := range c.setOptions {
		if i == option {
			return true
		}
	}

	return false
}

// NewConfig parses a config file and return the config
func NewConfig(path string) (*MCOConfig, error) {
	mcollective := newMcollective()
	mcollective.ConfigFile = path

	// TODO i think probably parse config can walk 'mcollective' recursively
	err := parseConfig(path, mcollective, "", &mcollective.setOptions)
	if err != nil {
		return nil, err
	}

	err = parseConfig(path, mcollective.Choria, "", &mcollective.setOptions)
	if err != nil {
		return nil, err
	}

	choriaPConf := filepath.Join(filepath.Dir(path), "plugin.d", "choria.cfg")
	if _, err := os.Stat(choriaPConf); err == nil {
		err = parseConfig(choriaPConf, mcollective.Choria, "plugin.choria", &mcollective.setOptions)
		if err != nil {
			return nil, err
		}
	}

	if mcollective.MainCollective == "" {
		mcollective.MainCollective = mcollective.Collectives[0]
	}

	// TODO other loglevels, not needed for this project
	if mcollective.LogLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	}

	return mcollective, nil
}

// parse a config file and fill in the given config structure based on its tags
func parseConfig(path string, config interface{}, prefix string, found *[]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	parseConfigContents(file, config, prefix, found)

	return nil
}

func parseConfigContents(content io.Reader, config interface{}, prefix string, found *[]string) {
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
				*found = append(*found, key)
			}
		}
	}
}

func newMcollective() *MCOConfig {
	m := &MCOConfig{Choria: newChoria()}
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
