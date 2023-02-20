// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package inter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/protocol"
	election "github.com/choria-io/go-choria/providers/election/streams"
	governor "github.com/choria-io/go-choria/providers/governor/streams"
	"github.com/choria-io/go-choria/providers/kv"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type ProtocolConstructor interface {
	NewMessage(payload []byte, agent string, collective string, msgType string, request Message) (msg Message, err error)
	NewMessageFromRequest(req protocol.Request, replyto string) (Message, error)
	NewReply(request protocol.Request) (reply protocol.Reply, err error)
	NewReplyFromMessage(version protocol.ProtocolVersion, msg Message) (rep protocol.Reply, err error)
	NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
	NewReplyTransportForMessage(msg Message, request protocol.Request) (protocol.TransportMessage, error)
	NewRequest(version protocol.ProtocolVersion, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error)
	NewRequestFromMessage(version protocol.ProtocolVersion, msg Message) (req protocol.Request, err error)
	NewRequestFromSecureRequest(sr protocol.SecureRequest) (request protocol.Request, err error)
	NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error)
	NewRequestID() (string, error)
	NewRequestMessageFromTransportJSON(payload []byte) (Message, error)
	NewRequestTransportForMessage(ctx context.Context, msg Message, version protocol.ProtocolVersion) (protocol.TransportMessage, error)
	NewSecureReply(reply protocol.Reply) (secure protocol.SecureReply, err error)
	NewSecureReplyFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureReply, err error)
	NewSecureRequest(ctx context.Context, request protocol.Request) (secure protocol.SecureRequest, err error)
	NewSecureRequestFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error)
	NewTransportForSecureReply(reply protocol.SecureReply) (message protocol.TransportMessage, err error)
	NewTransportForSecureRequest(request protocol.SecureRequest) (message protocol.TransportMessage, err error)
	NewTransportFromJSON(data []byte) (message protocol.TransportMessage, err error)
	NewTransportMessage(version protocol.ProtocolVersion) (message protocol.TransportMessage, err error)
	RequestProtocol() protocol.ProtocolVersion
}

type Framework interface {
	ProtocolConstructor
	ConfigurationProvider
	ConnectionManager

	BuildInfo() *build.Info
	CallerID() string
	Certname() string
	ClientTLSConfig() (*tls.Config, error)
	Colorize(c string, format string, a ...any) string
	ConfigureProvisioning(ctx context.Context)
	DDLResolvers() ([]DDLResolver, error)
	DisableTLSVerify() bool
	Enroll(ctx context.Context, wait time.Duration, cb func(digest string, try int)) error
	FacterCmd() string
	FacterDomain() (string, error)
	FacterFQDN() (string, error)
	FacterStringFact(fact string) (string, error)
	FederationCollectives() (collectives []string)
	FederationMiddlewareServers() (servers srvcache.Servers, err error)
	Getuid() int
	GovernorSubject(name string) string
	HTTPClient(secure bool) (*http.Client, error)
	HasCollective(collective string) bool
	IsFederated() (result bool)
	KV(ctx context.Context, conn Connector, bucket string, create bool, opts ...kv.Option) (nats.KeyValue, error)
	KVWithConn(ctx context.Context, conn Connector, bucket string, create bool, opts ...kv.Option) (nats.KeyValue, Connector, error)
	Logger(component string) *logrus.Entry
	MiddlewareServers() (servers srvcache.Servers, err error)
	NetworkBrokerPeers() (servers srvcache.Servers, err error)
	NewElection(ctx context.Context, conn Connector, name string, imported bool, opts ...election.Option) (Election, error)
	NewElectionWithConn(ctx context.Context, conn Connector, name string, imported bool, opts ...election.Option) (Election, Connector, error)
	NewGovernor(ctx context.Context, name string, conn Connector, opts ...governor.Option) (governor.Governor, Connector, error)
	NewGovernorManager(ctx context.Context, name string, limit uint64, maxAge time.Duration, replicas uint, update bool, conn Connector, opts ...governor.Option) (governor.Manager, Connector, error)
	OverrideCertname() string
	PQLQuery(query string) ([]byte, error)
	PQLQueryCertNames(query string) ([]string, error)
	ProgressWidth() int
	PrometheusTextFileDir() string
	ProvisionMode() bool
	ProvisioningServers(ctx context.Context) (srvcache.Servers, error)
	PublicCert() (*x509.Certificate, error)
	PuppetAIOCmd(command string, def string) string
	PuppetDBServers() (servers srvcache.Servers, err error)
	PuppetSetting(setting string) (string, error)
	QuerySrvRecords(records []string) (srvcache.Servers, error)
	SetLogWriter(out io.Writer)
	SetLogger(logger *logrus.Logger)
	SetupLogging(debug bool) (err error)
	SignerSeedFile() (f string, err error)
	SignerToken() (token string, exp time.Time, err error)
	SignerTokenFile() (f string, err error)
	SupportsProvisioning() bool
	TLSConfig() (*tls.Config, error)
	TrySrvLookup(names []string, defaultSrv srvcache.Server) (srvcache.Server, error)
	UniqueID() string
	UniqueIDFromUnverifiedToken() (id string, uid string, exp time.Time, token string, err error)
	ValidateSecurity() (errors []string, ok bool)
}
