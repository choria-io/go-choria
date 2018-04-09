package choria

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/srvcache"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// Framework is a utilty encompasing choria config and various utilities
type Framework struct {
	Config *Config

	mu    *sync.Mutex
	stats bool
}

// Server is a representation of a network server host and port
type Server struct {
	Host   string
	Port   int
	Scheme string
}

// URL creates a correct url from the server if scheme is known
func (self *Server) URL() (u *url.URL, err error) {
	if self.Scheme == "" {
		return u, fmt.Errorf("Server %s:%d has no scheme, cannot make a URL", self.Host, self.Port)
	}

	ustring := fmt.Sprintf("%s://%s:%d", self.Scheme, self.Host, self.Port)

	u, err = url.Parse(ustring)
	if err != nil {
		return u, fmt.Errorf("Could not parse %s: %s", ustring, err)
	}

	return
}

// New sets up a Choria with all its config loaded and so forth
func New(path string) (*Framework, error) {
	config, err := NewConfig(path)
	if err != nil {
		return nil, err
	}

	return NewWithConfig(config)
}

func NewWithConfig(config *Config) (*Framework, error) {
	c := Framework{
		Config: config,
		mu:     &sync.Mutex{},
	}

	if c.ProvisionMode() {
		c.ConfigureProvisioning()
	}

	err := c.SetupLogging(false)
	if err != nil {
		return &c, fmt.Errorf("Could not set up logging: %s", err)
	}

	if !config.DisableTLS {
		if errors, ok := c.CheckSSLSetup(); !ok {
			return &c, fmt.Errorf("SSL setup is not valid, %d errors encountered: %s", len(errors), strings.Join(errors, ", "))
		}
	}

	return &c, nil
}

// ProvisionMode determines if this instance is in provisioning mode
// if the setting `plugin.choria.server.provision` is set at all then
// the value of that is returned, else it the build time property
// ProvisionDefault is consulted
func (self *Framework) ProvisionMode() bool {
	if build.ProvisionBrokerURLs == "" {
		return false
	}

	if self.Config.HasOption("plugin.choria.server.provision") {
		return self.Config.Choria.Provision
	}

	return build.ProvisionDefault()
}

// ConfigureProvisioning adjusts the active configuration to match the
// provisioning profile
func (self *Framework) ConfigureProvisioning() {
	self.Config.Choria.FederationCollectives = []string{}
	self.Config.Collectives = []string{"provisioning"}
	self.Config.MainCollective = "provisioning"
	self.Config.RegisterInterval = 120
	self.Config.RegistrationSplay = false
	self.Config.Choria.FileContentRegistrationTarget = "choria.provisioning_data"
}

// IsFederated determiens if the configuration is setting up any Federation collectives
func (self *Framework) IsFederated() (result bool) {
	if len(self.FederationCollectives()) == 0 {
		return false
	}

	return true
}

// Logger creates a new logrus entry
func (self *Framework) Logger(component string) *log.Entry {
	return log.WithFields(log.Fields{"component": component})
}

// FederationCollectives determines the known Federation Member
// Collectives based on the CHORIA_FED_COLLECTIVE environment
// variable or the choria.federation.collectives config item
func (self *Framework) FederationCollectives() (collectives []string) {
	var found []string

	env := os.Getenv("CHORIA_FED_COLLECTIVE")

	if env != "" {
		found = strings.Split(env, ",")
	}

	if len(found) == 0 {
		found = self.Config.Choria.FederationCollectives
	}

	for _, collective := range found {
		collectives = append(collectives, strings.TrimSpace(collective))
	}

	return
}

// FederationMiddlewareServers determines the correct Federation Middleware Servers
//
// It does this by:
//
//    * looking for choria.federation_middleware_hosts configuration
//	  * Doing SRV lookups of  _mcollective-federation_server._tcp and _x-puppet-mcollective_federation._tcp
func (self *Framework) FederationMiddlewareServers() (servers []Server, err error) {
	configured := self.Config.Choria.FederationMiddlewareHosts
	if len(configured) > 0 {
		s, err := StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("Could not parse configured Federation Middleware: %s", err)
		}

		for _, server := range s {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		if servers, err = self.QuerySrvRecords([]string{"_mcollective-federation_server._tcp", "_x-puppet-mcollective_federation._tcp"}); err != nil {
			return servers, fmt.Errorf("Could not resolve Federation Middleware Server SRV records: %s", err)
		}
	}

	for i, s := range servers {
		s.Scheme = "nats"
		servers[i] = s
	}

	return
}

// ProvisioningServers determines the build time provisioning servers
// when it's unset or results in an empty server list this will return
// an error
func (self *Framework) ProvisioningServers() ([]Server, error) {
	if build.ProvisionBrokerURLs != "" {
		s := strings.Split(build.ProvisionBrokerURLs, ",")
		servers, err := StringHostsToServers(s, "nats")
		if err != nil {
			return servers, fmt.Errorf("Could not determine provisioning servers from %s: %s", build.ProvisionBrokerURLs, err)
		}

		if len(servers) == 0 {
			return servers, fmt.Errorf("ProvisionBrokerURLs '%s' is not in the correct format, 0 server:port combinations were found", build.ProvisionBrokerURLs)
		}

		return servers, nil
	}

	return []Server{}, fmt.Errorf("ProvisionBrokerURLs was not set during compile time")
}

// MiddlewareServers determines the correct Middleware Servers
//
// It does this by:
//
//    * looking for choria.federation_middleware_hosts configuration
//	  * Doing SRV lookups of _mcollective-server._tcp and __x-puppet-mcollective._tcp
//    * Defaulting to puppet:4222
func (self *Framework) MiddlewareServers() (servers []Server, err error) {
	if self.IsFederated() {
		return self.FederationMiddlewareServers()
	}

	configured := self.Config.Choria.MiddlewareHosts
	if len(configured) > 0 {
		s, err := StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("Could not parse configured Middleware: %s", err)
		}

		for _, server := range s {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		if servers, err = self.QuerySrvRecords([]string{"_mcollective-server._tcp", "_x-puppet-mcollective._tcp"}); err != nil {
			log.Warnf("Could not resolve Middleware Server SRV records: %s", err)
		}
	}

	if len(servers) == 0 {
		servers = []Server{Server{Host: "puppet", Port: 4222}}
	}

	for i, s := range servers {
		s.Scheme = "nats"
		servers[i] = s
	}

	return
}

// SetupLogging configures logging based on choria config directives
// currently only file and console behaviours are supported
func (self *Framework) SetupLogging(debug bool) (err error) {
	log.SetOutput(os.Stdout)

	if self.Config.LogFile != "" {
		log.SetFormatter(&log.JSONFormatter{})

		file, err := os.OpenFile(self.Config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("Could not set up logging: %s", err)
		}

		log.SetOutput(file)
	}

	switch self.Config.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	return
}

// TrySrvLookup will attempt to lookup a series of names returning the first found
// if SRV lookups are disabled or nothing is found the default will be returned
func (self *Framework) TrySrvLookup(names []string, defaultSrv Server) (Server, error) {
	if !self.Config.Choria.UseSRVRecords {
		return defaultSrv, nil
	}

	for _, q := range names {
		a, err := self.QuerySrvRecords([]string{q})
		if err == nil {
			log.Infof("Found %s:%d from %s SRV lookups", a[0].Host, a[0].Port, strings.Join(names, ", "))

			return a[0], nil
		}
	}

	log.Debugf("Could not find SRV records for %s, returning defaults %s:%d", strings.Join(names, ", "), defaultSrv.Host, defaultSrv.Port)

	return defaultSrv, nil
}

// QuerySrvRecords looks for SRV records within the right domain either
// thanks to facter domain or the configured domain.
//
// If the config disables SRV then a error is returned.
func (self *Framework) QuerySrvRecords(records []string) ([]Server, error) {
	servers := []Server{}

	if !self.Config.Choria.UseSRVRecords {
		return servers, errors.New("SRV lookups are disabled in the configuration file")
	}

	domain := self.Config.Choria.SRVDomain
	var err error

	if self.Config.Choria.SRVDomain == "" {
		domain, err = self.FacterDomain()
		if err != nil {
			return servers, err
		}

		// cache the result to speed things up
		self.Config.Choria.SRVDomain = domain
	}

	for _, q := range records {
		record := q + "." + domain
		log.Debugf("Attempting SRV lookup for %s", record)

		cname, addrs, err := srvcache.LookupSRV("", "", record, net.LookupSRV)
		if err != nil {
			log.Debugf("Failed to resolve %s: %s", record, err)
			continue
		}

		log.Debugf("Found %d SRV records for %s", len(addrs), cname)

		for _, a := range addrs {
			servers = append(servers, Server{Host: a.Target, Port: int(a.Port)})
		}
	}

	return servers, nil
}

// NetworkBrokerPeers are peers in the broker cluster resolved from
// _mcollective-broker._tcp or from the plugin config
func (self *Framework) NetworkBrokerPeers() (servers []Server, err error) {
	servers, err = self.QuerySrvRecords([]string{"_mcollective-broker._tcp"})
	if err != nil {
		log.Errorf("SRV lookup for _mcollective-broker._tcp failed: %s", err)
		err = nil
	}

	if len(servers) == 0 {
		for _, server := range self.Config.Choria.NetworkPeers {
			parsed, err := url.Parse(server)
			if err != nil {
				return servers, fmt.Errorf("Could not parse network peer %s: %s", server, err)
			}

			host, sport, err := net.SplitHostPort(parsed.Host)
			if err != nil {
				return servers, fmt.Errorf("Could not parse network peer %s: %s", server, err)
			}

			port, err := strconv.Atoi(sport)
			if err != nil {
				return servers, fmt.Errorf("Could not parse network peer %s: %s", server, err)
			}

			s := Server{
				Host:   host,
				Port:   port,
				Scheme: parsed.Scheme,
			}

			servers = append(servers, s)
		}
	}

	for i, s := range servers {
		s.Scheme = "nats"
		servers[i] = s
	}

	return
}

// DiscoveryServer is the server configured as a discovery proxy
func (self *Framework) DiscoveryServer() (Server, error) {
	s := Server{
		Host: self.Config.Choria.DiscoveryHost,
		Port: self.Config.Choria.DiscoveryPort,
	}

	if !self.ProxiedDiscovery() {
		return s, errors.New("Proxy discovery is not enabled")
	}

	result, err := self.TrySrvLookup([]string{"_mcollective-discovery._tcp"}, s)

	return result, err
}

// ProxiedDiscovery determines if a client is configured for proxied discover
func (self *Framework) ProxiedDiscovery() bool {
	if self.Config.HasOption("plugin.choria.discovery_host") || self.Config.HasOption("plugin.choria.discovery_port") {
		return true
	}

	return self.Config.Choria.DiscoveryProxy
}

// PuppetSetting retrieves a config setting by shelling out to puppet apply --configprint
func (self *Framework) PuppetSetting(setting string) (string, error) {
	return PuppetSetting(setting)
}

// FacterStringFact looks up a facter fact, returns "" when unknown
func (self *Framework) FacterStringFact(fact string) (string, error) {
	return FacterStringFact(fact)
}

// FacterFQDN determines the machines fqdn by querying facter.  Returns "" when unknown
func (self *Framework) FacterFQDN() (string, error) {
	return FacterStringFact("networking.fqdn")
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func (self *Framework) FacterDomain() (string, error) {
	return FacterStringFact("networking.domain")
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
func (self *Framework) FacterCmd() string {
	return PuppetAIOCmd("facter", "")
}

// PuppetAIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
//
// TODO: windows support
func (self *Framework) PuppetAIOCmd(command string, def string) string {
	return PuppetAIOCmd(command, def)
}

// NewRequestID Creates a new RequestID
func (self *Framework) NewRequestID() string {
	return strings.Replace(uuid.NewV4().String(), "-", "", -1)
}

// CallerID determines the cert based callerid
func (self *Framework) CallerID() string {
	return fmt.Sprintf("choria=%s", self.Certname())
}

// HasCollective determines if a collective is known in the configuration
func (self *Framework) HasCollective(collective string) bool {
	for _, c := range self.Config.Collectives {
		if c == collective {
			return true
		}
	}

	return false
}
