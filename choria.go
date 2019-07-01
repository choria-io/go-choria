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
	NatsCredentials          string   `confkey:"plugin.nats.credentials" environment:"MCOLLECTIVE_NATS_CREDENTIALS"`
	MiddlewareHosts          []string `confkey:"plugin.choria.middleware_hosts" type:"comma_split"`
	RandomizeMiddlewareHosts bool     `confkey:"plugin.choria.randomize_middleware_hosts" default:"true"`

	// network broker
	NetworkListenAddress      string        `confkey:"plugin.choria.network.listen_address" default:"::"`
	NetworkClientPort         int           `confkey:"plugin.choria.network.client_port" default:"4222"`
	NetworkPeerPort           int           `confkey:"plugin.choria.network.peer_port" default:"5222"`
	NetworkPeerUser           string        `confkey:"plugin.choria.network.peer_user"`
	NetworkPeerPassword       string        `confkey:"plugin.choria.network.peer_password"`
	NetworkPeers              []string      `confkey:"plugin.choria.network.peers" type:"comma_split"`
	NetworkWriteDeadline      time.Duration `confkey:"plugin.choria.network.write_deadline" type:"duration" default:"5s"`
	NetworkAllowedClientHosts []string      `confkey:"plugin.choria.network.client_hosts" type:"comma_split"`
	NetworkAccountOperator    string        `confkey:"plugin.choria.network.account_operator"`

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
	SecurityProvider             string   `confkey:"plugin.security.provider" default:"puppet" validate:"enum=puppet,file"`
	SecurityAlwaysOverwriteCache bool     `confkey:"plugin.security.always_overwrite_cache" default:"false"`

	// file security
	FileSecurityCertificate string `confkey:"plugin.security.file.certificate"`
	FileSecurityKey         string `confkey:"plugin.security.file.key"`
	FileSecurityCA          string `confkey:"plugin.security.file.ca"`
	FileSecurityCache       string `confkey:"plugin.security.file.cache"`

	// adapters
	Adapters []string `confkey:"plugin.choria.adapters" type:"comma_split"`

	// status file
	StatusFilePath      string `confkey:"plugin.choria.status_file_path"`
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
