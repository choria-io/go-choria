// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
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
	"github.com/choria-io/go-choria/providers/kv"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type ProtocolConstructor interface {
	NewMessage(payload string, agent string, collective string, msgType string, request Message) (msg Message, err error)
	NewMessageFromRequest(req protocol.Request, replyto string) (Message, error)
	NewReply(request protocol.Request) (reply protocol.Reply, err error)
	NewReplyFromMessage(version string, msg Message) (rep protocol.Reply, err error)
	NewReplyFromSecureReply(sr protocol.SecureReply) (reply protocol.Reply, err error)
	NewReplyFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Reply, err error)
	NewReplyTransportForMessage(msg Message, request protocol.Request) (protocol.TransportMessage, error)
	NewRequest(version string, agent string, senderid string, callerid string, ttl int, requestid string, collective string) (request protocol.Request, err error)
	NewRequestFromMessage(version string, msg Message) (req protocol.Request, err error)
	NewRequestFromSecureRequest(sr protocol.SecureRequest) (request protocol.Request, err error)
	NewRequestFromTransportJSON(payload []byte, skipvalidate bool) (msg protocol.Request, err error)
	NewRequestID() (string, error)
	NewRequestMessageFromTransportJSON(payload []byte) (Message, error)
	NewRequestTransportForMessage(msg Message, version string) (protocol.TransportMessage, error)
	NewSecureReply(reply protocol.Reply) (secure protocol.SecureReply, err error)
	NewSecureReplyFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureReply, err error)
	NewSecureRequest(request protocol.Request) (secure protocol.SecureRequest, err error)
	NewSecureRequestFromTransport(message protocol.TransportMessage, skipvalidate bool) (secure protocol.SecureRequest, err error)
	NewTransportForSecureReply(reply protocol.SecureReply) (message protocol.TransportMessage, err error)
	NewTransportForSecureRequest(request protocol.SecureRequest) (message protocol.TransportMessage, err error)
	NewTransportFromJSON(data string) (message protocol.TransportMessage, err error)
	NewTransportMessage(version string) (message protocol.TransportMessage, err error)
}

type Framework interface {
	ProtocolConstructor
	ConfigurationProvider
	ConnectionManager

	BuildInfo() *build.Info
	CallerID() string
	Certname() string
	ClientTLSConfig() (*tls.Config, error)
	Colorize(c string, format string, a ...interface{}) string
	ConfigureProvisioning()
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
	ShouldUseNGS() bool
	SignerToken() (token string, err error)
	SignerSeedFile() (f string, err error)
	SupportsProvisioning() bool
	TLSConfig() (*tls.Config, error)
	TrySrvLookup(names []string, defaultSrv srvcache.Server) (srvcache.Server, error)
	UniqueID() string
	UniqueIDFromUnverifiedToken() (caller string, id string, token string, err error)
	ValidateSecurity() (errors []string, ok bool)
}
