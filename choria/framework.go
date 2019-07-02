package choria

import (
	context "context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-protocol/protocol"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/provtarget"
	"github.com/choria-io/go-config"
	puppet "github.com/choria-io/go-puppet"
	"github.com/choria-io/go-security"
	"github.com/choria-io/go-security/filesec"
	"github.com/choria-io/go-security/puppetsec"
	"github.com/choria-io/go-srvcache"
	log "github.com/sirupsen/logrus"
)

// Framework is a utility encompassing choria config and various utilities
type Framework struct {
	Config *config.Config

	security security.Provider
	log      *logrus.Logger

	srvcache *srvcache.Cache
	puppet   *puppet.PuppetWrapper
	mu       *sync.Mutex
	stats    bool
}

// New sets up a Choria with all its config loaded and so forth
func New(path string) (*Framework, error) {
	conf, err := config.NewConfig(path)
	if err != nil {
		return nil, err
	}

	conf.ApplyBuildSettings(&build.Info{})

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

	c.srvcache = srvcache.New(cfg.Identity, 5*time.Second, net.LookupSRV, c.Logger("srvcache"))
	c.puppet = puppet.New()

	err = c.setupSecurity()
	if err != nil {
		return &c, fmt.Errorf("could not set up security framework: %s", err)
	}

	config.Mutate(cfg, c.Logger("config"))

	return &c, nil
}

// BuildInfo retrieves build information
func (fw *Framework) BuildInfo() *build.Info {
	return &build.Info{}
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
func (fw *Framework) FederationMiddlewareServers() (servers srvcache.Servers, err error) {
	configured := fw.Config.Choria.FederationMiddlewareHosts
	servers = srvcache.NewServers()

	if len(configured) > 0 {
		servers, err = srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured Federation Middleware: %s", err)
		}
	}

	if servers.Count() == 0 {
		servers, err = fw.QuerySrvRecords([]string{"_mcollective-federation_server._tcp", "_x-puppet-mcollective_federation._tcp"})
		if err != nil {
			return servers, fmt.Errorf("could not resolve Federation Middleware Server SRV records: %s", err)
		}
	}

	servers.Each(func(s srvcache.Server) {
		if s.Scheme() == "" {
			s.SetScheme("nats")
		}
	})

	return servers, err
}

// ProvisioningServers determines the build time provisioning servers
// when it's unset or results in an empty server list this will return
// an error
func (fw *Framework) ProvisioningServers(ctx context.Context) (srvcache.Servers, error) {
	return provtarget.Targets(ctx, fw.Logger("provtarget"))
}

// MiddlewareServers determines the correct Middleware Servers
//
// It does this by:
//
//    * looking for choria.federation_middleware_hosts configuration
//	  * Doing SRV lookups of _mcollective-server._tcp and __x-puppet-mcollective._tcp
//    * Defaulting to puppet:4222
func (fw *Framework) MiddlewareServers() (servers srvcache.Servers, err error) {
	if fw.IsFederated() {
		return fw.FederationMiddlewareServers()
	}

	servers = srvcache.NewServers()
	configured := fw.Config.Choria.MiddlewareHosts

	if len(configured) > 0 {
		servers, err = srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured Middleware: %s", err)
		}
	}

	if servers.Count() == 0 {
		servers, err = fw.QuerySrvRecords([]string{"_mcollective-server._tcp", "_x-puppet-mcollective._tcp"})
		if err != nil {
			log.Warnf("Could not resolve Middleware Server SRV records: %s", err)
		}
	}

	if servers.Count() == 0 {
		servers = srvcache.NewServers(srvcache.NewServer("puppet", 4222, "nats"))
	}

	servers.Each(func(s srvcache.Server) {
		if s.Scheme() == "" {
			s.SetScheme("nats")
		}
	})

	return servers, nil
}

// SetupLogging configures logging based on choria config directives
// currently only file and console behaviors are supported
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
		if err == nil && a.Count() > 0 {
			found := a.Servers()[0]
			log.Infof("Found %s:%d from %s SRV lookups", found.Host(), found.Port(), strings.Join(names, ", "))

			return found, nil
		}
	}

	log.Debugf("Could not find SRV records for %s, returning defaults %s:%d", strings.Join(names, ", "), defaultSrv.Host(), defaultSrv.Port())

	return defaultSrv, nil
}

// QuerySrvRecords looks for SRV records within the right domain either
// thanks to facter domain or the configured domain.
//
// If the config disables SRV then a error is returned.
func (fw *Framework) QuerySrvRecords(records []string) (srvcache.Servers, error) {
	servers := srvcache.NewServers()

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

		servers, err = fw.srvcache.LookupSrvServers("", "", record, "")
		if err != nil {
			log.Debugf("Failed to resolve %s: %s", record, err)
			continue
		}

		log.Debugf("Found %d SRV records for %s", servers.Count(), record)
		break
	}

	return servers, nil
}

// NetworkBrokerPeers are peers in the broker cluster resolved from
// _mcollective-broker._tcp or from the plugin config
func (fw *Framework) NetworkBrokerPeers() (servers srvcache.Servers, err error) {
	servers, err = fw.QuerySrvRecords([]string{"_mcollective-broker._tcp"})
	if err != nil {
		log.Errorf("SRV lookup for _mcollective-broker._tcp failed: %s", err)
		err = nil
	}

	if servers.Count() == 0 {
		servers, err = srvcache.StringHostsToServers(fw.Config.Choria.NetworkPeers, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse network peers: %s", err)
		}
	}

	servers.Each(func(f srvcache.Server) {
		if f.Scheme() == "" {
			f.SetScheme("nats")
		}
	})

	return servers, nil
}

// DiscoveryServer is the server configured as a discovery proxy
func (fw *Framework) DiscoveryServer() (srvcache.Server, error) {
	dflt := srvcache.NewServer(fw.Config.Choria.DiscoveryHost, fw.Config.Choria.DiscoveryPort, "")

	if !fw.ProxiedDiscovery() {
		return dflt, errors.New("Proxy discovery is not enabled")
	}

	return fw.TrySrvLookup([]string{"_mcollective-discovery._tcp"}, dflt)
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
	return fw.puppet.Setting(setting)
}

// FacterStringFact looks up a facter fact, returns "" when unknown
func (fw *Framework) FacterStringFact(fact string) (string, error) {
	return fw.puppet.FacterStringFact(fact)
}

// FacterFQDN determines the machines fqdn by querying facter.  Returns "" when unknown
func (fw *Framework) FacterFQDN() (string, error) {
	return fw.puppet.FacterStringFact("networking.fqdn")
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func (fw *Framework) FacterDomain() (string, error) {
	return fw.puppet.FacterStringFact("networking.domain")
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
func (fw *Framework) FacterCmd() string {
	return fw.puppet.AIOCmd("facter", "")
}

// PuppetAIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
func (fw *Framework) PuppetAIOCmd(command string, def string) string {
	return fw.puppet.AIOCmd(command, def)
}

// NewRequestID Creates a new RequestID
func (fw *Framework) NewRequestID() (string, error) {
	return NewRequestID()
}

// UniqueID creates a new unique ID, usually a v4 uuid, if that fails a random string based ID is made
func (fw *Framework) UniqueID() string {
	return UniqueID()
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
