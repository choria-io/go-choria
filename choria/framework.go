// Copyright (c) 2017-2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package choria

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/providers/ddlresolver"
	election "github.com/choria-io/go-choria/providers/election/streams"
	governor "github.com/choria-io/go-choria/providers/governor/streams"
	"github.com/choria-io/go-choria/providers/kv"
	"github.com/choria-io/go-choria/providers/provtarget"
	"github.com/choria-io/go-choria/providers/security/choria"
	"github.com/choria-io/go-choria/providers/signers"
	"github.com/choria-io/tokens"
	"github.com/fatih/color"
	"github.com/nats-io/nats.go"
	"github.com/segmentio/ksuid"
	"golang.org/x/term"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/protocol"
	certmanagersec "github.com/choria-io/go-choria/providers/security/certmanager"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"

	"github.com/choria-io/go-choria/providers/security/filesec"
	"github.com/choria-io/go-choria/providers/security/puppetsec"
	"github.com/choria-io/go-choria/puppet"
	"github.com/choria-io/go-choria/srvcache"
	log "github.com/sirupsen/logrus"
)

// Framework is a utility encompassing choria config and various utilities
type Framework struct {
	Config *config.Config

	inProcessConnection nats.InProcessConnProvider
	customRequestSigner inter.RequestSigner
	security            inter.SecurityProvider
	log                 *log.Logger

	bi       *build.Info
	srvcache *srvcache.Cache
	puppet   *puppet.Wrapper
	mu       *sync.Mutex
}

type Option func(fw *Framework) error

// WithCustomRequestSigner sets a custom request signer, generally only used in tests
func WithCustomRequestSigner(s inter.RequestSigner) Option {
	return func(fw *Framework) error {
		fw.customRequestSigner = s
		return nil
	}
}

// New sets up a Choria with all its config loaded and so forth
func New(path string, opts ...Option) (*Framework, error) {
	conf, err := config.NewConfig(path)
	if err != nil {
		return nil, err
	}

	conf.ApplyBuildSettings(BuildInfo())

	return NewWithConfig(conf, opts...)
}

// NewWithConfig creates a new instance of the framework with the supplied config instance
func NewWithConfig(cfg *config.Config, opts ...Option) (*Framework, error) {
	c := Framework{
		Config: cfg,
		mu:     &sync.Mutex{},
		bi:     BuildInfo(),
	}

	for _, opt := range opts {
		err := opt(&c)
		if err != nil {
			return nil, err
		}
	}

	err := c.SetupLogging(false)
	if err != nil {
		return &c, fmt.Errorf("could not set up logging: %s", err)
	}

	config.Mutate(cfg, c.Logger("config"))

	c.srvcache = srvcache.New(cfg.Identity, 5*time.Second, net.LookupSRV, c.Logger("srvcache"))
	c.puppet = puppet.New()

	err = c.setupSecurity()
	if err != nil {
		return &c, fmt.Errorf("could not set up security framework: %s", err)
	}

	return &c, nil
}

// SetInProcessConnProvider sets a nats.InProcessConnProvider to use, connector will make connections using that if set
func (fw *Framework) SetInProcessConnProvider(p nats.InProcessConnProvider) {
	fw.mu.Lock()
	fw.inProcessConnection = p
	fw.mu.Unlock()
}

// InProcessConn provides an in-process connection for nats if configured using SetInProcessConnProvider()
func (fw *Framework) InProcessConn() (net.Conn, error) {
	fw.mu.Lock()
	ipc := fw.inProcessConnection
	fw.mu.Unlock()

	if ipc == nil {
		return nil, fmt.Errorf("invalid connection")
	}

	return ipc.InProcessConn()
}

// BuildInfo retrieves build information
func (fw *Framework) BuildInfo() *build.Info {
	return BuildInfo()
}

func (fw *Framework) setupSecurity() error {
	var (
		err    error
		signer inter.RequestSigner
	)

	switch {
	case fw.customRequestSigner != nil:
		signer = fw.customRequestSigner
	case fw.Config.Choria.RemoteSignerService:
		signer = signers.NewAAAServiceRPCSigner(fw)
	case fw.Config.Choria.RemoteSignerURL != "":
		signer = signers.NewAAAServiceHTTPSigner()
	}

	switch fw.Config.Choria.SecurityProvider {
	case "puppet":
		fw.security, err = puppetsec.New(
			puppetsec.WithResolver(fw),
			puppetsec.WithChoriaConfig(fw.BuildInfo(), fw.Config),
			puppetsec.WithLog(fw.Logger("security")),
			puppetsec.WithSigner(signer))

	case "file":
		fw.security, err = filesec.New(
			filesec.WithChoriaConfig(fw.BuildInfo(), fw.Config),
			filesec.WithLog(fw.Logger("security")),
			filesec.WithSigner(signer))

	case "pkcs11":
		err = fw.setupPKCS11(signer)

	case "certmanager":
		fw.security, err = certmanagersec.New(
			certmanagersec.WithChoriaConfig(fw.Config),
			certmanagersec.WithLog(fw.Logger("security")),
			certmanagersec.WithContext(context.Background()))

	case "choria":
		fw.security, err = choria.New(
			choria.WithChoriaConfig(fw.Config),
			choria.WithLog(fw.Logger("security")),
			choria.WithSigner(signer))

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
	if !fw.Config.InitiatedByServer || (fw.bi.ProvisionBrokerURLs() == "" && fw.bi.ProvisionJWTFile() == "" && fw.bi.ProvisionToken() == "") {
		return false
	}

	if fw.Config.HasOption("plugin.choria.server.provision") {
		return fw.Config.Choria.Provision
	}

	// some build environments might go through provisioning as a test phase
	// and if those base images do not remove their config they might start up
	// with still-valid tokens from building phase instead of re-provisioning
	// as intended, typically in those cases the hostnames will have changed between
	// base image and eventual running OS.  So we can detect that by comparing identity
	// of the provisioned token and the running identity.
	//
	// Regardless how it happened this is bad because the node will subscribe to its
	// node subject with its identity set but the token will have another identity in
	// it causing the broker to not set appropriate allow rules for that connection
	caller, _, _, _, _ := fw.UniqueIDFromUnverifiedToken()
	if caller != "" {
		if fw.Config.Identity != caller {
			fw.log.Warnf("Caller %q from server token does not match identity %q, enabling provisioning", caller, fw.Config.Identity)
			return true
		}
	}

	return fw.bi.ProvisionDefault()
}

// PrometheusTextFileDir is the configured directory where to write prometheus text file stats
func (fw *Framework) PrometheusTextFileDir() string {
	return fw.Config.Choria.PrometheusTextFileDir
}

// SupportsProvisioning determines if a node can auto provision
func (fw *Framework) SupportsProvisioning() bool {
	if fw.ProvisionMode() {
		return true
	}

	return fw.bi.SupportsProvisioning()
}

// ConfigureProvisioning adjusts the active configuration to match the provisioning profile
func (fw *Framework) ConfigureProvisioning(ctx context.Context) {
	provtarget.Configure(ctx, fw.Config, fw.Logger("provtarget"))

	if !fw.ProvisionMode() {
		return
	}

	fw.Config.RPCAuthorization = false
	fw.Config.Choria.FederationCollectives = []string{}
	fw.Config.Collectives = []string{"provisioning"}
	fw.Config.MainCollective = "provisioning"
	fw.Config.Registration = []string{}
	fw.Config.FactSourceFile = fw.bi.ProvisionFacts()
	fw.Config.Choria.NatsUser = fw.bi.ProvisioningBrokerUsername()
	fw.Config.Choria.NatsPass = fw.bi.ProvisioningBrokerPassword()
	fw.Config.Choria.ProvisionAllowUpdate = fw.bi.ProvisionAllowServerUpdate()
	fw.Config.Choria.SSLDir = filepath.Join(filepath.Dir(fw.Config.ConfigFile), "ssl")
	fw.Config.Choria.SecurityProvider = "file"

	if fw.bi.ProvisionStatusFile() != "" {
		fw.Config.Choria.StatusFilePath = fw.bi.ProvisionStatusFile()
		fw.Config.Choria.StatusUpdateSeconds = 10
	}

	if fw.bi.ProvisionRegistrationData() != "" {
		fw.Config.RegistrationCollective = "provisioning"
		fw.Config.Registration = []string{"file_content"}
		fw.Config.RegisterInterval = 120
		fw.Config.RegistrationSplay = false
		fw.Config.Choria.FileContentRegistrationTarget = "provisioning.registration.data"
		fw.Config.Choria.FileContentRegistrationData = fw.bi.ProvisionRegistrationData()
	}

	if !fw.bi.ProvisionSecurity() {
		protocol.Secure = "false"
	}

	if fw.bi.ProvisionBrokerSRVDomain() != "" {
		fw.Config.Choria.UseSRVRecords = true
		fw.Config.Choria.SRVDomain = fw.bi.ProvisionBrokerSRVDomain()
	}

	if fw.bi.ProvisionJWTFile() != "" && fw.bi.ProvisionUsingVersion2() {
		fw.Config.Choria.SecurityProvider = "choria"
		fw.Config.Choria.ChoriaSecurityTokenFile = fw.bi.ProvisionJWTFile()
		fw.Config.Choria.ChoriaSecuritySignReplies = false
		protocol.Secure = "false"

		err := fw.setupSecurity()
		if err != nil {
			fw.log.Errorf("Could not setup security to enable protocol v2: %v", err)
		}
	}
}

// IsFederated determines if the configuration is setting up any Federation collectives
func (fw *Framework) IsFederated() (result bool) {
	return len(fw.FederationCollectives()) != 0
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
//   - looking for choria.federation_middleware_hosts configuration
//   - Doing SRV lookups of  _mcollective-federation_server._tcp and _x-puppet-mcollective_federation._tcp
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

// PuppetDBServers resolves the PuppetDB server based on configuration of _x-puppet-db._tcp
func (fw *Framework) PuppetDBServers() (servers srvcache.Servers, err error) {
	if fw.Config.Choria.PuppetDBHost != "" {
		configured := fmt.Sprintf("%s:%d", fw.Config.Choria.PuppetDBHost, fw.Config.Choria.PuppetDBPort)

		servers, err = srvcache.StringHostsToServers([]string{configured}, "https")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured PuppetDB host: %s", err)
		}

		return servers, nil
	}

	if fw.Config.Choria.UseSRVRecords {
		servers, err = fw.QuerySrvRecords([]string{"_x-puppet-db._tcp"})
		if err != nil {
			return servers, fmt.Errorf("could not resolve PuppetDB Server SRV records: %s", err)
		}

		if servers.Count() == 0 {
			servers, err = fw.QuerySrvRecords([]string{"_x-puppet._tcp"})
			if err != nil {
				return servers, fmt.Errorf("could not resolve Puppet Server SRV records: %s", err)
			}

			// In the case where we take _x-puppet._tcp SRV records we unfortunately have
			// to force the port else it uses the one from Puppet which will 404
			servers.Each(func(s srvcache.Server) {
				s.SetPort(fw.Config.Choria.PuppetDBPort)
			})
		}

		servers.Each(func(s srvcache.Server) {
			if s.Scheme() == "" {
				s.SetScheme("https")
			}
		})
	}

	if servers == nil || servers.Count() == 0 {
		configured := fmt.Sprintf("%s:%d", "puppet", fw.Config.Choria.PuppetDBPort)

		servers, err = srvcache.StringHostsToServers([]string{configured}, "https")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured PuppetDB host: %s", err)
		}
	}

	return servers, nil
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
//   - if ngs is configured and credentials are set and middleware_hosts are empty, use ngs
//   - looking for choria.federation_middleware_hosts configuration
//   - Doing SRV lookups of _mcollective-server._tcp and __x-puppet-mcollective._tcp
//   - Defaulting to puppet:4222
func (fw *Framework) MiddlewareServers() (servers srvcache.Servers, err error) {
	configured := fw.Config.Choria.MiddlewareHosts

	if fw.IsFederated() {
		return fw.FederationMiddlewareServers()
	}

	servers = srvcache.NewServers()

	if len(configured) > 0 {
		servers, err = srvcache.StringHostsToServers(configured, "nats")
		if err != nil {
			return servers, fmt.Errorf("could not parse configured Middleware: %s", err)
		}
	}

	if servers.Count() == 0 && fw.Config.Choria.UseSRVRecords {
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

func (fw *Framework) SetLogWriter(out io.Writer) {
	if fw.log != nil {
		fw.log.SetOutput(out)
	}
}

func (fw *Framework) commonLogOpener() error {
	switch {
	case strings.ToLower(fw.Config.LogFile) == "discard":
		fw.log.SetOutput(io.Discard)

	case strings.ToLower(fw.Config.LogFile) == "stdout":
		fw.log.SetOutput(os.Stdout)

	case strings.ToLower(fw.Config.LogFile) == "stderr":
		fw.log.SetOutput(os.Stderr)

	case fw.Config.LogFile != "":
		fw.log.Formatter = &log.JSONFormatter{}

		file, err := os.OpenFile(fw.Config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("could not set up logging: %s", err)
		}

		fw.log.SetOutput(file)
	}

	return nil
}

// SetLogger sets the logger to use
func (fw *Framework) SetLogger(logger *log.Logger) {
	fw.log = logger
}

// SetupLogging configures logging based on choria config directives
// currently only file and console behaviors are supported
func (fw *Framework) SetupLogging(debug bool) (err error) {
	if fw.Config.CustomLogger != nil {
		fw.log = fw.Config.CustomLogger
		return
	}

	fw.log = log.New()
	fw.log.SetOutput(os.Stdout)

	err = fw.openLogfile()
	if err != nil {
		return err
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

// TrySrvLookup will attempt to look up a series of names returning the first found
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

var errSRVDisabled = errors.New("SRV lookups are disabled in the configuration file")

// QuerySrvRecords looks for SRV records within the right domain either
// thanks to facter domain or the configured domain.
//
// If the config disables SRV then a error is returned.
func (fw *Framework) QuerySrvRecords(records []string) (srvcache.Servers, error) {
	servers := srvcache.NewServers()

	if !fw.Config.Choria.UseSRVRecords {
		return servers, errSRVDisabled
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
		if !errors.Is(err, errSRVDisabled) {
			fw.log.Errorf("SRV lookup for _mcollective-broker._tcp failed: %s", err)
		}

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
	if fw.RequestProtocol() == protocol.RequestV2 {
		kid, err := ksuid.NewRandom()
		if err != nil {
			return "", err
		}
		return kid.String(), nil
	}

	return strings.Replace(util.UniqueID(), "-", "", -1), nil
}

// UniqueID creates a new unique ID, usually a v4 uuid, if that fails a random string based ID is made
func (fw *Framework) UniqueID() string {
	return util.UniqueID()
}

// CallerID determines the cert based callerid
func (fw *Framework) CallerID() string {
	caller, _, _, _, err := fw.UniqueIDFromUnverifiedToken()
	if err == nil {
		return caller
	}

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

// UniqueIDFromUnverifiedToken extracts the caller id or identity from a token, the token is not verified as we do not have the certificate
func (fw *Framework) UniqueIDFromUnverifiedToken() (id string, uid string, exp time.Time, token string, err error) {
	ts, exp, err := fw.SignerToken()
	if err != nil {
		return "", "", exp, "", err
	}

	if fw.Config.InitiatedByServer {
		t, id, err := tokens.UnverifiedIdentityFromServerToken(ts)
		if err != nil {
			return "", "", exp, "", err
		}

		return id, fmt.Sprintf("%x", md5.Sum([]byte(id))), exp, t.Raw, nil
	} else {
		t, caller, err := tokens.UnverifiedCallerFromClientIDToken(ts)
		if err != nil {
			return "", "", exp, "", err
		}

		return caller, fmt.Sprintf("%x", md5.Sum([]byte(caller))), exp, t.Raw, nil
	}
}

// SignerSeedFile is the path to the seed file for JWT auth
// TODO: we need to revisit the many ways to set a seed file here and try to come up with fewer options (1740)
func (fw *Framework) SignerSeedFile() (f string, err error) {
	switch {
	case fw.Config.Choria.ChoriaSecuritySeedFile != "":
		return fw.Config.Choria.ChoriaSecuritySeedFile, nil
	case fw.Config.Choria.ServerAnonTLS:
		if fw.Config.Choria.ServerTokenSeedFile != "" {
			return fw.Config.Choria.ServerTokenSeedFile, nil
		}
	case fw.Config.Choria.RemoteSignerTokenSeedFile != "":
		return fw.Config.Choria.RemoteSignerTokenSeedFile, nil
	}

	t, err := fw.SignerTokenFile()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.key", strings.TrimSuffix(t, filepath.Ext(t))), nil
}

// SignerTokenFile is the path to the token file, supports clients and servers
// TODO: we need to revisit the many ways to set a token file here and try to come up with fewer options (1740)
func (fw *Framework) SignerTokenFile() (f string, err error) {
	tf := ""

	switch {
	case fw.Config.Choria.ChoriaSecurityTokenFile != "":
		tf = fw.Config.Choria.ChoriaSecurityTokenFile
	case fw.Config.Choria.RemoteSignerTokenFile != "":
		tf = fw.Config.Choria.RemoteSignerTokenFile
	case fw.Config.Choria.ServerAnonTLS:
		tf = fw.Config.Choria.ServerTokenFile
	}

	if tf == "" {
		return "", fmt.Errorf("no token file defined")
	}

	return tf, nil
}

// SignerToken retrieves the token used for signing requests or connecting to the broker
func (fw *Framework) SignerToken() (token string, expiry time.Time, err error) {
	var exp time.Time

	tf, err := fw.SignerTokenFile()
	if err != nil {
		return "", exp, err
	}

	tb, err := os.ReadFile(tf)
	if err != nil {
		return "", exp, fmt.Errorf("could not read token file: %v", err)
	}

	purpose := tokens.TokenPurpose(string(tb))
	switch purpose {
	case tokens.ClientIDPurpose:
		claims, err := tokens.ParseClientIDTokenUnverified(string(tb))
		if err != nil {
			return "", exp, err
		}
		err = claims.Valid()
		if err != nil {
			fw.log.Warnf("Authentication token %s is not valid: %v", tf, err)
			return "", exp, err
		}
		exp = claims.ExpireTime()

	case tokens.ServerPurpose:
		claims, err := tokens.ParseServerTokenUnverified(string(tb))
		if err != nil {
			return "", exp, err
		}
		err = claims.Valid()
		if err != nil {
			fw.log.Warnf("Authentication token %s is not valid: %v", tf, err)
			return "", exp, err
		}
		exp = claims.ExpireTime()

	case tokens.ProvisioningPurpose:
		// nothing to verify here

	default:
		return "", exp, fmt.Errorf("cannot use token %s with purpose %q as signer token", tf, purpose)
	}

	return strings.TrimSpace(string(tb)), exp, err
}

// HTTPClient creates a *http.Client prepared by the security provider with certificates and more set
func (fw *Framework) HTTPClient(secure bool) (*http.Client, error) {
	return fw.security.HTTPClient(secure)
}

func (fw *Framework) PQLQuery(query string) ([]byte, error) {
	q := url.Values{}
	q.Set("query", query)
	path := fmt.Sprintf("/pdb/query/v4?%s", q.Encode())

	pdb, err := fw.PuppetDBServers()
	if err != nil {
		return nil, err
	}
	pdbhost := pdb.Strings()[0]

	fw.log.Debugf("Performing PQL query against %s: %s", pdbhost, query)

	client, err := fw.HTTPClient(true)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", fmt.Sprintf("%s%s", pdbhost, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid PuppetDB response: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (fw *Framework) PQLQueryCertNames(query string) ([]string, error) {
	body, err := fw.PQLQuery(query)
	if err != nil {
		return nil, err
	}

	var res []struct {
		Certname    string `json:"certname"`
		Deactivated bool   `json:"deactivated"`
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, r := range res {
		if !r.Deactivated {
			nodes = append(nodes, r.Certname)
		}
	}

	return nodes, nil
}

// Colorize returns a string of either 'red', 'green' or 'yellow'. If the 'color' configuration
// is set to false then the string will have no color hints
func (fw *Framework) Colorize(c string, format string, a ...any) string {
	if !fw.Config.Color {
		return fmt.Sprintf(format, a...)
	}

	switch c {
	case "red":
		return color.RedString(fmt.Sprintf(format, a...))
	case "green":
		return color.GreenString(fmt.Sprintf(format, a...))
	case "yellow":
		return color.YellowString(fmt.Sprintf(format, a...))
	default:
		return fmt.Sprintf(format, a...)
	}
}

// ProgressWidth determines the width of the progress bar, when -1 there is not enough space for a progress bar
func (fw *Framework) ProgressWidth() int {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}

	if width < 35 {
		return -1
	}

	width -= 30
	if width > 80 {
		width = 80
	}

	return width
}

// GovernorSubject the subject to use for choria managed Governors
func (fw *Framework) GovernorSubject(name string) string {
	return util.GovernorSubject(name, fw.Config.MainCollective)
}

// NewGovernor creates a new governor client with its own connection when none is given
func (fw *Framework) NewGovernor(ctx context.Context, name string, conn inter.Connector, opts ...governor.Option) (governor.Governor, inter.Connector, error) {
	var err error
	if conn == nil {
		conn, err = fw.NewConnector(ctx, fw.MiddlewareServers, fmt.Sprintf("governor client: %s", name), fw.Logger("governor"))
		if err != nil {
			return nil, nil, err
		}
	}

	return governor.New(name, conn.Nats(), opts...), conn, nil
}

// NewGovernorManager creates a new governor manager with its own connection when none is given
func (fw *Framework) NewGovernorManager(ctx context.Context, name string, limit uint64, maxAge time.Duration, replicas uint, update bool, conn inter.Connector, opts ...governor.Option) (governor.Manager, inter.Connector, error) {
	var err error

	if conn == nil {
		conn, err = fw.NewConnector(ctx, fw.MiddlewareServers, fmt.Sprintf("governor manager: %s", name), fw.Logger("governor"))
		if err != nil {
			return nil, nil, err
		}
	}

	gov, err := governor.NewManager(name, limit, maxAge, replicas, conn.Nats(), update, opts...)
	if err != nil {
		return nil, nil, err
	}

	return gov, conn, nil
}

// NewElection establishes a new, named, leader election requiring a Choria Streams bucket called CHORIA_LEADER_ELECTION.
// This will create a new network connection per election, see NewElectionWithConn() to re-use an existing connection
func (fw *Framework) NewElection(ctx context.Context, conn inter.Connector, name string, imported bool, opts ...election.Option) (inter.Election, error) {
	e, _, err := fw.NewElectionWithConn(ctx, conn, name, imported, opts...)

	return e, err
}

// NewElectionWithConn establish a new, named, leader election requiring a Choria Streams bucket called CHORIA_LEADER_ELECTION.
func (fw *Framework) NewElectionWithConn(ctx context.Context, conn inter.Connector, name string, imported bool, opts ...election.Option) (inter.Election, inter.Connector, error) {
	var err error

	if conn == nil {
		conn, err = fw.NewConnector(ctx, fw.MiddlewareServers, fmt.Sprintf("election %s %s", name, fw.Config.Identity), fw.Logger("election"))
		if err != nil {
			return nil, nil, err
		}
	}

	var jsopt []nats.JSOpt
	if imported {
		jsopt = append(jsopt, nats.APIPrefix("choria.streams"))
	}

	js, err := conn.Nats().JetStream(jsopt...)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to Choria Streams: %s", err)
	}

	kv, err := js.KeyValue("CHORIA_LEADER_ELECTION")
	if err != nil {
		return nil, nil, fmt.Errorf("cannot access KV Bucket CHORIA_LEADER_ELECTION")
	}

	e, err := election.NewElection(fw.Config.Identity, name, kv, opts...)
	if err != nil {
		return nil, nil, err
	}

	return e, conn, nil
}

// KV creates a connection to a key-value store and gives access to the connector
func (fw *Framework) KV(ctx context.Context, conn inter.Connector, bucket string, create bool, opts ...kv.Option) (nats.KeyValue, error) {
	kv, _, err := fw.KVWithConn(ctx, conn, bucket, create, opts...)
	return kv, err
}

// KVWithConn creates a connection to a key-value store and gives access to the connector
func (fw *Framework) KVWithConn(ctx context.Context, conn inter.Connector, bucket string, create bool, opts ...kv.Option) (nats.KeyValue, inter.Connector, error) {
	logger := fw.Logger("kv")

	var err error

	if conn == nil {
		conn, err = fw.NewConnector(ctx, fw.MiddlewareServers, fmt.Sprintf("kv %s", fw.CallerID()), logger)
		if err != nil {
			return nil, nil, err
		}
	}

	b, err := kv.NewKV(conn.Nats(), bucket, create, opts...)
	if err != nil {
		return nil, nil, err
	}

	return b, conn, err
}

func (fw *Framework) DDLResolvers() ([]inter.DDLResolver, error) {
	resolvers := []inter.DDLResolver{
		&ddlresolver.InternalCachedDDLResolver{},
		&ddlresolver.FileSystemDDLResolver{},
	}

	if fw.Config.Choria.RegistryClientCache != "" {
		resolvers = append(resolvers, &ddlresolver.RegistryDDLResolver{})
	}

	return resolvers, nil
}
