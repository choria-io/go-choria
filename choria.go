package config

import (
	"time"

	confkey "github.com/choria-io/go-confkey"
	log "github.com/sirupsen/logrus"
)

// ChoriaPluginConfig settings
type ChoriaPluginConfig struct {
	PuppetServerHost string `confkey:"plugin.choria.puppetserver_host" default:"puppet"`
	PuppetServerPort int    `confkey:"plugin.choria.puppetserver_port" default:"8140"`
	PuppetCAHost     string `confkey:"plugin.choria.puppetca_host" default:"puppet"`
	PuppetCAPort     int    `confkey:"plugin.choria.puppetca_port" default:"8140"`
	PuppetDBHost     string `confkey:"plugin.choria.puppetdb_host" default:"puppet"`
	PuppetDBPort     int    `confkey:"plugin.choria.puppetdb_port" default:"8081"`
	SSLDir           string `confkey:"plugin.choria.ssldir" type:"path_string"`
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

	StatsListenAddress    string `confkey:"plugin.choria.stats_address" default:"127.0.0.1"`
	StatsPort             int    `confkey:"plugin.choria.stats_port" default:"0"`
	LegacyLifeCycleFormat bool   `confkey:"plugin.choria.legacy_lifecycle_format" default:"0"`

	// nats connector
	NatsUser                 string   `confkey:"plugin.nats.user" environment:"MCOLLECTIVE_NATS_USERNAME"`
	NatsPass                 string   `confkey:"plugin.nats.pass" environment:"MCOLLECTIVE_NATS_PASSWORD"`
	NatsCredentials          string   `confkey:"plugin.nats.credentials" environment:"MCOLLECTIVE_NATS_CREDENTIALS"`
	NatsNGS                  bool     `confkey:"plugin.nats.ngs" environment:"MCOLLECTIVE_NATS_NGS"`
	MiddlewareHosts          []string `confkey:"plugin.choria.middleware_hosts" type:"comma_split"`
	RandomizeMiddlewareHosts bool     `confkey:"plugin.choria.randomize_middleware_hosts" default:"true"`

	// network broker
	NetworkListenAddress      string        `confkey:"plugin.choria.network.listen_address" default:"::"`
	NetworkClientPort         int           `confkey:"plugin.choria.network.client_port" default:"4222"`
	NetworkClientTLSForce     bool          `confkey:"plugin.choria.network.client_tls_force_required"`
	NetworkPeerPort           int           `confkey:"plugin.choria.network.peer_port" default:"5222"`
	NetworkPeerUser           string        `confkey:"plugin.choria.network.peer_user"`
	NetworkPeerPassword       string        `confkey:"plugin.choria.network.peer_password"`
	NetworkPeers              []string      `confkey:"plugin.choria.network.peers" type:"comma_split"`
	NetworkLeafPort           int           `confkey:"plugin.choria.network.leafnode_port" default:"0"`
	NetworkLeafRemotes        []string      `confkey:"plugin.choria.network.leafnode_remotes" type:"comma_split"`
	NetworkGatewayPort        int           `confkey:"plugin.choria.network.gateway_port" default:"0"`
	NetworkGatewayName        string        `confkey:"plugin.choria.network.gateway_name" default:"CHORIA"`
	NetworkGatewayRemotes     []string      `confkey:"plugin.choria.network.gateway_remotes" type:"comma_split"`
	NetworkWriteDeadline      time.Duration `confkey:"plugin.choria.network.write_deadline" type:"duration" default:"5s"`
	NetworkAllowedClientHosts []string      `confkey:"plugin.choria.network.client_hosts" type:"comma_split"`
	NetworkAccountOperator    string        `confkey:"plugin.choria.network.operator_account"`
	NetworkSystemAccount      string        `confkey:"plugin.choria.network.system_account"`
	NetworkTLSTimeout         int           `confkey:"plugin.choria.network.tls_timeout" default:"2"`

	// broker features
	BrokerNetwork    bool `confkey:"plugin.choria.broker_network" default:"false"`
	BrokerDiscovery  bool `confkey:"plugin.choria.broker_discovery" default:"false"`
	BrokerFederation bool `confkey:"plugin.choria.broker_federation" default:"false"`

	// registration
	FileContentRegistrationData   string `confkey:"plugin.choria.registration.file_content.data" default:""`
	FileContentRegistrationTarget string `confkey:"plugin.choria.registration.file_content.target" default:""`
	FileContentCompression        bool   `confkey:"plugin.choria.registration.file_content.compression" default:"true"`

	// ruby compatibility
	RubyAgentShim   string   `confkey:"plugin.choria.agent_provider.mcorpc.agent_shim"`
	RubyAgentConfig string   `confkey:"plugin.choria.agent_provider.mcorpc.config"`
	RubyLibdir      []string `confkey:"plugin.choria.agent_provider.mcorpc.libdir" type:"path_split"`

	// security plugin
	PrivilegedUsers              []string `confkey:"plugin.choria.security.privileged_users" type:"comma_split" default:"\\.privileged.mcollective$,\\.privileged.choria$"`
	CertnameWhitelist            []string `confkey:"plugin.choria.security.certname_whitelist" type:"comma_split" default:"\\.mcollective$,\\.choria$"`
	Serializer                   string   `confkey:"plugin.choria.security.serializer" validate:"enum=json,yaml"`
	SecurityProvider             string   `confkey:"plugin.security.provider" default:"puppet" validate:"enum=puppet,file,pkcs11"`
	SecurityAlwaysOverwriteCache bool     `confkey:"plugin.security.always_overwrite_cache" default:"false"`
	RemoteSignerTokenFile        string   `confkey:"plugin.choria.security.request_signer.token_file" type:"path_string"`
	RemoteSignerTokenEnvironment string   `confkey:"plugin.choria.security.request_signer.token_environment"`
	RemoteSignerURL              string   `confkey:"plugin.choria.security.request_signer.url"`

	// file security
	FileSecurityCertificate string   `confkey:"plugin.security.file.certificate" type:"path_string"`
	FileSecurityKey         string   `confkey:"plugin.security.file.key" type:"path_string"`
	FileSecurityCA          string   `confkey:"plugin.security.file.ca" type:"path_string"`
	FileSecurityCache       string   `confkey:"plugin.security.file.cache" type:"path_string"`

	// TLS Parameters
	CipherSuites            []string `confkey:"plugin.security.cipher_suites" type:"comma_split"`
	ECCCurves               []string `confkey:"plugin.security.ecc_curves", type:"comma_split"`

	// pkcs11 security
	PKCS11DriverFile string `confkey:"plugin.security.pkcs11.driver_file" type:"path_string"`
	PKCS11Slot       int    `confkey:"plugin.security.pkcs11.slot"`

	// adapters
	Adapters []string `confkey:"plugin.choria.adapters" type:"comma_split"`

	// status file
	StatusFilePath      string `confkey:"plugin.choria.status_file_path" type:"path_string"`
	StatusUpdateSeconds int    `confkey:"plugin.choria.status_update_interval" default:"30"`

	// machine
	MachineSourceDir string `confkey:"plugin.choria.machine.store"`
}

func newChoria() *ChoriaPluginConfig {
	c := &ChoriaPluginConfig{}

	err := confkey.SetStructDefaults(c)
	if err != nil {
		log.Errorf("Choria config creation failed: %s", err)
	}

	return c
}
