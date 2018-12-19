package choria

import (
	context "context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-protocol/protocol"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/provtarget"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/choria-io/go-security"
	"github.com/choria-io/go-security/filesec"
	"github.com/choria-io/go-security/puppetsec"
	log "github.com/sirupsen/logrus"
)

// Framework is a utilty encompasing choria config and various utilities
type Framework struct {
	Config *config.Config

	security security.Provider
	log      *logrus.Logger

	mu    *sync.Mutex
	stats bool
}

// New sets up a Choria with all its config loaded and so forth
func New(path string) (*Framework, error) {
	conf, err := config.NewConfig(path)
	if err != nil {
		return nil, err
	}

	return NewWithConfig(conf)
}

// NewWithConfig creates a new instance of the framework with the supplied config instance
func NewWithConfig(cfg *config.Config) (*Framework, error) {
	c := Framework{
		Config: cfg,
		mu:     &sync.Mutex{},
	}

	if c.ProvisionMode() {
		c.ConfigureProvisioning()
	}

	err := c.SetupLogging(false)
	if err != nil {
		return &c, fmt.Errorf("could not set up logging: %s", err)
	}

	err = c.setupSecurity()
	if err != nil {
		return &c, fmt.Errorf("could not set up security framework: %s", err)
	}

	config.Mutate(cfg, c.Logger("config"))

	return &c, nil
}

func (fw *Framework) setupSecurity() error {
	var err error

	switch fw.Config.Choria.SecurityProvider {
	case "puppet":
		fw.security, err = puppetsec.New(puppetsec.WithResolver(fw), puppetsec.WithChoriaConfig(fw.Config), puppetsec.WithLog(fw.Logger("security")))
	case "file":
		fw.security, err = filesec.New(filesec.WithChoriaConfig(fw.Config), filesec.WithLog(fw.Logger("security")))
	default:
		err = fmt.Errorf("unknown security provider %s", fw.Config.Choria.SecurityProvider)
	}

	if err != nil {
		return err
	}

	if !(fw.Config.DisableSecurityProviderVerify || fw.Config.DisableTLS) && protocol.IsSecure() {
		errors, ok := fw.security.Validate()
		if !ok {
			return fmt.Errorf("security setup is not valid, %d errors encountered: %s", len(errors), strings.Join(errors, ", "))
		}
	}

	return nil
}

// ProvisionMode determines if this instance is in provisioning mode
// if the setting `plugin.choria.server.provision` is set at all then
// the value of that is returned, else it the build time property
// ProvisionDefault is consulted
func (fw *Framework) ProvisionMode() bool {
	if !fw.Config.InitiatedByServer || build.ProvisionBrokerURLs == "" {
		return false
	}

	if fw.Config.HasOption("plugin.choria.server.provision") {
		return fw.Config.Choria.Provision
	}

	return build.ProvisionDefault()
}

// ConfigureProvisioning adjusts the active configuration to match the
// provisioning profile
func (fw *Framework) ConfigureProvisioning() {
	fw.Config.Choria.FederationCollectives = []string{}
	fw.Config.Collectives = []string{"provisioning"}
	fw.Config.MainCollective = "provisioning"
	fw.Config.Registration = []string{}
	fw.Config.FactSourceFile = build.ProvisionFacts

	if build.ProvisionStatusFile != "" {
		fw.Config.Choria.StatusFilePath = build.ProvisionStatusFile
		fw.Config.Choria.StatusUpdateSeconds = 10
	}

	if build.ProvisionRegistrationData != "" {
		fw.Config.RegistrationCollective = "provisioning"
		fw.Config.Registration = []string{"file_content"}
		fw.Config.RegisterInterval = 120
		fw.Config.RegistrationSplay = false
		fw.Config.Choria.FileContentRegistrationTarget = "choria.provisioning_data"
		fw.Config.Choria.FileContentRegistrationData = build.ProvisionRegistrationData
	}

	if !build.ProvisionSecurity() {
		protocol.Secure = "false"
		fw.Config.Choria.SecurityProvider = "file"
		fw.Config.DisableTLS = true
	}
}

// IsFederated determiens if the configuration is setting up any Federation collectives
func (fw *Framework) IsFederated() (result bool) {
	if len(fw.FederationCollectives()) == 0 {
		return false
	}

	return true
}

// Logger creates a new logrus entry
func (fw *Framework) Logger(component string) *log.Entry {
	return fw.log.WithFields(log.Fields{"component": component})
}

// FederationCollectives determines the known Federation Member
// Collectives based on the CHORIA_FED_COLLECTIVE environment
// variable or the choria.federation.collectives config item
func (fw *Framework) FederationCollectives() (collectives []string) {
	var found []string

	env := os.Getenv("CHORIA_FED_COLLECTIVE")

	if env != "" {
		found = strings.Split(env, ",")
	}

	if len(found) == 0 {
		found = fw.Config.Choria.FederationCollectives
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
func (fw *Framework) FederationMiddlewareServers() (servers []srvcache.Server, err error) {
	configured := fw.Config.Choria.FederationMiddlewareHosts
	if len(configured) > 0 {
		s, err := srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("Could not parse configured Federation Middleware: %s", err)
		}

		for _, server := range s {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		if servers, err = fw.QuerySrvRecords([]string{"_mcollective-federation_server._tcp", "_x-puppet-mcollective_federation._tcp"}); err != nil {
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
func (fw *Framework) ProvisioningServers(ctx context.Context) ([]srvcache.Server, error) {
	return provtarget.Targets(ctx, fw.Logger("provtarget"))
}

// MiddlewareServers determines the correct Middleware Servers
//
// It does this by:
//
//    * looking for choria.federation_middleware_hosts configuration
//	  * Doing SRV lookups of _mcollective-server._tcp and __x-puppet-mcollective._tcp
//    * Defaulting to puppet:4222
func (fw *Framework) MiddlewareServers() (servers []srvcache.Server, err error) {
	if fw.IsFederated() {
		return fw.FederationMiddlewareServers()
	}

	configured := fw.Config.Choria.MiddlewareHosts
	if len(configured) > 0 {
		s, err := srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("Could not parse configured Middleware: %s", err)
		}

		for _, server := range s {
			servers = append(servers, server)
		}
	}

	if len(servers) == 0 {
		if servers, err = fw.QuerySrvRecords([]string{"_mcollective-server._tcp", "_x-puppet-mcollective._tcp"}); err != nil {
			log.Warnf("Could not resolve Middleware Server SRV records: %s", err)
		}
	}

	if len(servers) == 0 {
		servers = []srvcache.Server{srvcache.Server{Host: "puppet", Port: 4222}}
	}

	for i, s := range servers {
		s.Scheme = "nats"
		servers[i] = s
	}

	return
}

// SetupLogging configures logging based on choria config directives
// currently only file and console behaviours are supported
func (fw *Framework) SetupLogging(debug bool) (err error) {
	fw.log = log.New()

	fw.log.Out = os.Stdout

	if fw.Config.LogFile != "" {
		fw.log.Formatter = &log.JSONFormatter{}

		file, err := os.OpenFile(fw.Config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("Could not set up logging: %s", err)
		}

		fw.log.Out = file
	}

	switch fw.Config.LogLevel {
	case "debug":
		fw.log.SetLevel(log.DebugLevel)
	case "info":
		fw.log.SetLevel(log.InfoLevel)
	case "warn":
		fw.log.SetLevel(log.WarnLevel)
	case "error":
		fw.log.SetLevel(log.ErrorLevel)
	case "fatal":
		fw.log.SetLevel(log.FatalLevel)
	default:
		fw.log.SetLevel(log.WarnLevel)
	}

	if debug {
		fw.log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(fw.log.Formatter)
	log.SetLevel(fw.log.Level)
	log.SetOutput(fw.log.Out)

	return
}

// TrySrvLookup will attempt to lookup a series of names returning the first found
// if SRV lookups are disabled or nothing is found the default will be returned
func (fw *Framework) TrySrvLookup(names []string, defaultSrv srvcache.Server) (srvcache.Server, error) {
	if !fw.Config.Choria.UseSRVRecords {
		return defaultSrv, nil
	}

	for _, q := range names {
		a, err := fw.QuerySrvRecords([]string{q})
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
func (fw *Framework) QuerySrvRecords(records []string) ([]srvcache.Server, error) {
	servers := []srvcache.Server{}

	if !fw.Config.Choria.UseSRVRecords {
		return servers, errors.New("SRV lookups are disabled in the configuration file")
	}

	domain := fw.Config.Choria.SRVDomain
	var err error

	if fw.Config.Choria.SRVDomain == "" {
		domain, err = fw.FacterDomain()
		if err != nil {
			return servers, err
		}

		// cache the result to speed things up
		fw.Config.Choria.SRVDomain = domain
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
			servers = append(servers, srvcache.Server{Host: strings.TrimRight(a.Target, "."), Port: int(a.Port)})
		}
	}

	return servers, nil
}

// NetworkBrokerPeers are peers in the broker cluster resolved from
// _mcollective-broker._tcp or from the plugin config
func (fw *Framework) NetworkBrokerPeers() (servers []srvcache.Server, err error) {
	servers, err = fw.QuerySrvRecords([]string{"_mcollective-broker._tcp"})
	if err != nil {
		log.Errorf("SRV lookup for _mcollective-broker._tcp failed: %s", err)
		err = nil
	}

	if len(servers) == 0 {
		for _, server := range fw.Config.Choria.NetworkPeers {
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

			s := srvcache.Server{
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
func (fw *Framework) DiscoveryServer() (srvcache.Server, error) {
	s := srvcache.Server{
		Host: fw.Config.Choria.DiscoveryHost,
		Port: fw.Config.Choria.DiscoveryPort,
	}

	if !fw.ProxiedDiscovery() {
		return s, errors.New("Proxy discovery is not enabled")
	}

	result, err := fw.TrySrvLookup([]string{"_mcollective-discovery._tcp"}, s)

	return result, err
}

// ProxiedDiscovery determines if a client is configured for proxied discover
func (fw *Framework) ProxiedDiscovery() bool {
	if fw.Config.HasOption("plugin.choria.discovery_host") || fw.Config.HasOption("plugin.choria.discovery_port") {
		return true
	}

	return fw.Config.Choria.DiscoveryProxy
}

// Getuid returns the numeric user id of the caller
func (fw *Framework) Getuid() int {
	return os.Getuid()
}

// PuppetSetting retrieves a config setting by shelling out to puppet apply --configprint
func (fw *Framework) PuppetSetting(setting string) (string, error) {
	return PuppetSetting(setting)
}

// FacterStringFact looks up a facter fact, returns "" when unknown
func (fw *Framework) FacterStringFact(fact string) (string, error) {
	return FacterStringFact(fact)
}

// FacterFQDN determines the machines fqdn by querying facter.  Returns "" when unknown
func (fw *Framework) FacterFQDN() (string, error) {
	return FacterStringFact("networking.fqdn")
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func (fw *Framework) FacterDomain() (string, error) {
	return FacterStringFact("networking.domain")
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
func (fw *Framework) FacterCmd() string {
	return PuppetAIOCmd("facter", "")
}

// PuppetAIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
func (fw *Framework) PuppetAIOCmd(command string, def string) string {
	return PuppetAIOCmd(command, def)
}

// NewRequestID Creates a new RequestID
func (fw *Framework) NewRequestID() (string, error) {
	return NewRequestID()
}

// CallerID determines the cert based callerid
func (fw *Framework) CallerID() string {
	return fmt.Sprintf("choria=%s", fw.Certname())
}

// HasCollective determines if a collective is known in the configuration
func (fw *Framework) HasCollective(collective string) bool {
	for _, c := range fw.Config.Collectives {
		if c == collective {
			return true
		}
	}

	return false
}

// OverrideCertname indicates if the user wish to force a specific certname, empty when not
func (fw *Framework) OverrideCertname() string {
	return fw.Config.OverrideCertname
}

// DisableTLSVerify indicates if the user whish to disable TLS verification
func (fw *Framework) DisableTLSVerify() bool {
	return fw.Config.DisableTLSVerify
}

// Configuration returns the active configuration
func (fw *Framework) Configuration() *config.Config {
	return fw.Config
}
