package mcollective

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// Choria is a utilty encompasing mcollective and choria config and various utilities
type Choria struct {
	Config *MCOConfig
}

// Server is a representation of a network server host and port
type Server struct {
	Host   string
	Port   int
	Scheme string
}

// URL creates a correct url from the server if scheme is known
func (s *Server) URL() (u *url.URL, err error) {
	if s.Scheme == "" {
		return u, fmt.Errorf("Server %s:%d has no scheme, cannot make a URL", s.Host, s.Port)
	}

	ustring := fmt.Sprintf("%s://%s:%d", s.Scheme, s.Host, s.Port)

	u, err = url.Parse(ustring)
	if err != nil {
		return u, fmt.Errorf("Could not parse %s: %s", ustring, err.Error())
	}

	return
}

// New sets up a Choria with all its config loaded and so forth
func New(path string) (*Choria, error) {
	c := Choria{}

	config, err := NewConfig(path)
	if err != nil {
		return &c, err
	}

	c.Config = config

	if errors, ok := c.CheckSSLSetup(); !ok {
		err = fmt.Errorf("SSL setup is not valid, %d errors encountered: %s", len(errors), strings.Join(errors, ", "))
		return &c, err
	}

	return &c, nil
}

// IsFederated determiens if the configuration is setting up any Federation collectives
func (c *Choria) IsFederated() (result bool) {
	if len(c.FederationCollectives()) == 0 {
		return false
	}

	return true
}

// FederationCollectives determines the known Federation Member
// Collectives based on the CHORIA_FED_COLLECTIVE environment
// variable or the choria.federation.collectives config item
func (c *Choria) FederationCollectives() (collectives []string) {
	var found []string

	env := os.Getenv("CHORIA_FED_COLLECTIVE")

	if env != "" {
		found = strings.Split(env, ",")
	}

	if len(found) == 0 {
		found = c.Config.Choria.FederationCollectives
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
func (c *Choria) FederationMiddlewareServers() (servers []Server, err error) {
	configured := c.Config.Choria.FederationMiddlewareHosts
	if len(configured) > 0 {
		s, err := StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("Could not parse configured Federation Middleware: %s", err.Error())
		}

		for _, server := range s {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		if servers, err = c.QuerySrvRecords([]string{"_mcollective-server._tcp", "_x-puppet-mcollective._tcp"}); err != nil {
			return servers, fmt.Errorf("Could not resolve Federation Middleware Server SRV records: %s", err.Error())
		}
	}

	for _, s := range servers {
		s.Scheme = "nats"
	}

	return
}

// SetupLogging configures logging based on mcollective config directives
// currently only file and console behaviours are supported
func (c *Choria) SetupLogging(debug bool) (err error) {
	log.SetOutput(os.Stdout)

	if c.Config.LogFile != "" {
		log.SetFormatter(&log.JSONFormatter{})

		file, err := os.OpenFile(c.Config.LogFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("Could not set up logging: %s", err.Error())
		}

		log.SetOutput(file)
	}

	switch c.Config.LogLevel {
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
func (c *Choria) TrySrvLookup(names []string, defaultSrv Server) (Server, error) {
	if !c.Config.Choria.UseSRVRecords {
		return defaultSrv, nil
	}

	for _, q := range names {
		a, err := c.QuerySrvRecords([]string{q})
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
func (c *Choria) QuerySrvRecords(records []string) ([]Server, error) {
	servers := []Server{}

	if !c.Config.Choria.UseSRVRecords {
		return servers, errors.New("SRV lookups are disabled in the configuration file")
	}

	domain, err := c.FacterDomain()
	if err != nil {
		return servers, err
	}

	for _, q := range records {
		record := q + "." + domain
		log.Debugf("Attempting SRV lookup for %s", record)

		cname, addrs, err := net.LookupSRV(record, "", "")
		if err != nil {
			return servers, err
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
func (c *Choria) NetworkBrokerPeers() (servers []Server, err error) {
	servers, err = c.QuerySrvRecords([]string{"_mcollective-broker._tcp"})
	if err != nil {
		log.Errorf("SRV lookup for _mcollective-broker._tcp failed: %s", err.Error())
		err = nil
	}

	if len(servers) == 0 {
		for _, server := range c.Config.Choria.NetworkPeers {
			parsed, err := url.Parse(server)
			if err != nil {
				return servers, fmt.Errorf("Could not parse network peer %s: %s", server, err.Error())
			}

			host, sport, err := net.SplitHostPort(parsed.Host)
			if err != nil {
				return servers, fmt.Errorf("Could not parse network peer %s: %s", server, err.Error())
			}

			port, err := strconv.Atoi(sport)
			if err != nil {
				return servers, fmt.Errorf("Could not parse network peer %s: %s", server, err.Error())
			}

			s := Server{
				Host:   host,
				Port:   port,
				Scheme: parsed.Scheme,
			}

			servers = append(servers, s)
		}
	}

	for _, s := range servers {
		s.Scheme = "nats"
	}

	return
}

// DiscoveryServer is the server configured as a discovery proxy
func (c *Choria) DiscoveryServer() (Server, error) {
	s := Server{
		Host: c.Config.Choria.DiscoveryHost,
		Port: c.Config.Choria.DiscoveryPort,
	}

	if !c.ProxiedDiscovery() {
		return s, errors.New("Proxy discovery is not enabled")
	}

	result, err := c.TrySrvLookup([]string{"_mcollective-discovery._tcp"}, s)

	return result, err
}

// ProxiedDiscovery determines if a client is configured for proxied discover
func (c *Choria) ProxiedDiscovery() bool {
	if c.Config.HasOption("plugin.choria.discovery_host") || c.Config.HasOption("plugin.choria.discovery_port") {
		return true
	}

	return c.Config.Choria.DiscoveryProxy
}

// PuppetSetting retrieves a config setting by shelling out to puppet apply --configprint
func (c *Choria) PuppetSetting(setting string) (string, error) {
	args := []string{"apply", "--configprint", setting}

	out, err := exec.Command("puppet", args...).Output()
	if err != nil {
		return "", err
	}

	return strings.Replace(string(out), "\n", "", -1), nil
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func (c *Choria) FacterDomain() (string, error) {
	cmd := c.FacterCmd()

	if cmd == "" {
		return "", errors.New("Could not find your facter command")
	}

	out, err := exec.Command(cmd, "networking.domain").Output()
	if err != nil {
		return "", errors.New("Could not resolve the server domain via facter: " + err.Error())
	}

	return strings.Replace(string(out), "\n", "", -1), nil
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
// TODO: windows support
func (c *Choria) FacterCmd() string {
	if _, err := os.Stat("/opt/puppetlabs/bin/facter"); err == nil {
		return "/opt/puppetlabs/bin/facter"
	}

	path, err := exec.LookPath("facter")
	if err != nil {
		return ""
	}

	return path
}

// NewRequestID Creates a new RequestID
func (c *Choria) NewRequestID() string {
	return strings.Replace(uuid.NewV4().String(), "-", "", -1)
}

// CallerID determines the cert based callerid
func (c *Choria) CallerID() string {
	return fmt.Sprintf("choria=%s", c.Certname())
}

// HasCollective determines if a collective is known in the configuration
func (c Choria) HasCollective(collective string) bool {
	for _, c := range c.Config.Collectives {
		if c == collective {
			return true
		}
	}

	return false
}
