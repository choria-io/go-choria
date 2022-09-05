// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"crypto/ed25519"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/sirupsen/logrus"
)

// ChoriaAuth implements the Nats server.Authentication interface and
// allows IP limits to be configured, connections that do not match
// the configured IP or CIDRs are not allowed to publish to the
// network targets used by clients to request actions on nodes.
//
// Additionally, when the server is running in a mode where anonymous
// TLS connections is accepted then servers are entirely denied and
// clients are allowed but restricted based on the JWT issued by the
// AAA Service. This is activated using the plugin.choria.network.client_anon_tls
// setting, however this should be avoided atm.
//
// Clients can present a JWT token signed by the AAA service if that
// token has a purpose field matching choria_client_id and if the
// AAA signer is configured in the broker using plugin.choria.security.request_signing_certificate
// those with valid tokens and that are fully verified can connect but
// will be restricted to client only functions. These clients will not
// be able to access any Choria Streams features, KV buckets etc
//
// Additionally when provisioning support is enabled any non mTLS connection
// will be placed in the provisioning account and unable to connect to the
// fleet or provisioned nodes. This is only enabled if plugin.choria.network.provisioning.signer_cert
// is set
type ChoriaAuth struct {
	clientAllowList         []string
	isTLS                   bool
	denyServers             bool
	provisioningTokenSigner string
	clientJwtSigner         string
	serverJwtSigner         string
	choriaAccount           *server.Account
	systemAccount           *server.Account
	provisioningAccount     *server.Account
	provPass                string
	systemUser              string
	systemPass              string
	log                     *logrus.Entry
}

const (
	provisioningUser = "provisioner"
	emptyString      = ""
)

var allSubjects = []string{">"}

// Check checks and registers the incoming connection
func (a *ChoriaAuth) Check(c server.ClientAuthentication) bool {
	var (
		verified    bool
		tlsVerified bool
		err         error
	)

	tlsc := c.GetTLSConnectionState()
	if tlsc != nil {
		tlsVerified = len(tlsc.VerifiedChains) > 0
	}

	if a.isTLS && tlsc == nil {
		a.log.Warnf("Did not receive TLS Connection State for connection %s, rejecting", c.RemoteAddress())
		return false
	}

	log := a.log.WithField("stage", "check")

	remote := c.RemoteAddress()
	if remote != nil {
		log = log.WithField("remote", remote.String())
	}

	systemUser := a.isSystemUser(c)
	pipeConnection := remote.String() == "pipe"

	switch {
	case a.isProvisionUser(c):
		verified, err = a.handleProvisioningUserConnection(c, tlsVerified)
		if err != nil {
			log.Warnf("Handling provisioning user connection failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

	case systemUser && (tlsVerified || pipeConnection):
		verified, err = a.handleVerifiedSystemAccount(c, log)
		if err != nil {
			log.Warnf("Handling system user failed, denying: %s", err)
		}

	case systemUser && tlsc == nil:
		verified = false
		log.Warnf("System user is only allowed over TLS connections")

	case systemUser && !tlsVerified:
		verified, err = a.handleUnverifiedSystemAccount(c, tlsc, log)
		if err != nil {
			log.Warnf("Handling unverified TLS system user failed, denying: %s", err)
		}

	default:
		verified, err = a.handleDefaultConnection(c, tlsc, tlsVerified, log)
		if err != nil {
			log.Warnf("Handling normal connection failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

		if !verified && a.isTLS && !tlsVerified {
			verified, err = a.handleUnverifiedProvisioningConnection(c)
			if err != nil {
				log.Warnf("Handling unverified connection failed, denying %s: %s", c.RemoteAddress().String(), err)
			}
		}
	}

	// should be already but let's make sure
	if err != nil {
		verified = false
	}

	return verified
}

func (a *ChoriaAuth) verifyNonceSignature(nonce []byte, sig string, pks string, log *logrus.Entry) (bool, error) {
	if sig == "" {
		return false, fmt.Errorf("connection nonce was not signed")
	}

	if pks == "" {
		return false, fmt.Errorf("no public key found in the JWT to verify nonce signature")
	}

	if len(nonce) == 0 {
		return false, fmt.Errorf("server did not generate a nonce to verify")
	}

	pubK, err := hex.DecodeString(pks)
	if err != nil {
		return false, fmt.Errorf("invalid nonce signature")
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return false, fmt.Errorf("invalid url encoded signature: %s", err)
	}

	valid, err := a.ed25519Verify(pubK, nonce, sigBytes)
	if err != nil {
		return false, fmt.Errorf("could not verify nonce signature: %v", err)
	}

	if !valid {
		return false, fmt.Errorf("nonce signature did not verify using pub key in the jwt")
	}

	log.Debugf("Successfully verified nonce signature")

	return true, nil
}

// ed25519.Verify() panics on bad pubkeys, this does not
func (a *ChoriaAuth) ed25519Verify(publicKey ed25519.PublicKey, message []byte, sig []byte) (bool, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key length %d", len(publicKey))
	}

	return ed25519.Verify(publicKey, message, sig), nil
}

func (a *ChoriaAuth) verifyServerJWTBasedAuth(remote net.Addr, jwts string, nonce []byte, sig string, log *logrus.Entry) (claims *tokens.ServerClaims, err error) {
	if remote == nil {
		log.Errorf("no remote client information received")
		return nil, fmt.Errorf("remote client information is required in anonymous TLS or JWT signing modes")
	}

	claims, err = a.parseServerJWT(jwts)
	if err != nil {
		log.Errorf("could not parse JWT from %s: %s", remote.String(), err)
		return nil, fmt.Errorf("invalid JWT token")
	}

	_, err = a.verifyNonceSignature(nonce, sig, claims.PublicKey, log)
	if err != nil {
		log.Errorf("nonce signature verification failed: %s", err)
		return nil, fmt.Errorf("invalid nonce signature")
	}

	return claims, nil
}

func (a *ChoriaAuth) verifyClientJWTBasedAuth(remote net.Addr, jwts string, nonce []byte, sig string, log *logrus.Entry) (claims *tokens.ClientIDClaims, err error) {
	if remote == nil {
		log.Errorf("no remote connection details received")
		return nil, fmt.Errorf("remote client information is required in anonymous TLS or JWT signing modes")
	}

	claims, err = a.parseClientIDJWT(jwts)
	if err != nil {
		log.Errorf("could not parse JWT from %s: %s", remote.String(), err)
		return nil, fmt.Errorf("invalid JWT token")
	}

	_, err = a.verifyNonceSignature(nonce, sig, claims.PublicKey, log)
	if err != nil {
		log.Errorf("nonce signature verification failed: %s", err)
		return nil, fmt.Errorf("invalid nonce signature")
	}

	return claims, nil
}

func (a *ChoriaAuth) handleDefaultConnection(c server.ClientAuthentication, conn *tls.ConnectionState, tlsVerified bool, log *logrus.Entry) (bool, error) {
	user := a.createUser(c)
	remote := c.RemoteAddress()
	opts := c.GetOpts()
	nonce := c.GetNonce()
	jwts := opts.Token
	caller := ""
	pipeConnection := remote.String() == "pipe"

	var err error

	log = log.WithField("mTLS", tlsVerified)

	if tlsVerified && len(conn.PeerCertificates) > 0 {
		log = log.WithField("subject", conn.PeerCertificates[0].Subject.CommonName)
	}
	if user.Account != nil {
		log = log.WithField("account", user.Account.Name)
	}

	var (
		serverClaims   *tokens.ServerClaims
		clientClaims   *tokens.ClientIDClaims
		setClientPerms bool
		setServerPerms bool
	)

	shouldPerformJWTBasedAuth := jwts != emptyString && conn != nil

	if shouldPerformJWTBasedAuth {
		purpose := tokens.TokenPurpose(jwts)
		log = log.WithFields(logrus.Fields{"jwt_auth": shouldPerformJWTBasedAuth, "purpose": purpose})
		log.Infof("Performing JWT based authentication verification")

		switch purpose {
		case tokens.ClientIDPurpose:
			if c.Kind() != server.CLIENT {
				return false, fmt.Errorf("a client JWT was presented by a %d connection", c.Kind())
			}

			clientClaims, err = a.verifyClientJWTBasedAuth(remote, jwts, nonce, opts.Sig, log)
			if err != nil {
				return false, fmt.Errorf("invalid nonce signature or jwt token")
			}
			log = log.WithField("caller", caller)
			log.Debugf("Extracted caller id %s from JWT token", clientClaims.CallerID)

			caller = clientClaims.CallerID
			setClientPerms = true
			user.Username = caller

		case tokens.ServerPurpose:
			if c.Kind() != server.CLIENT {
				return false, fmt.Errorf("a server JWT was presented by a %d connection", c.Kind())
			}

			serverClaims, err = a.verifyServerJWTBasedAuth(remote, jwts, nonce, opts.Sig, log)
			if err != nil {
				return false, fmt.Errorf("invalid nonce signature or jwt token")
			}
			log = log.WithField("identity", serverClaims.ChoriaIdentity)
			log.Debugf("Extracted remote identity %s from JWT token", serverClaims.ChoriaIdentity)

			setServerPerms = true
			user.Username = serverClaims.ChoriaIdentity

		default:
			return false, fmt.Errorf("do not know how to handle %v purpose token", purpose)
		}
	}

	switch {
	case !shouldPerformJWTBasedAuth && !tlsVerified && !pipeConnection:
		log.Warnf("Rejecting unverified connection without token")
		return false, fmt.Errorf("unverified connection without JWT token")

	// if a caller is set from the Client ID JWT we want to restrict it to just client stuff
	// if a client allow list is set and the client is in the ip range we restrict it also
	// else its default open like users with certs
	case setClientPerms || (!setServerPerms && caller != "" && a.remoteInClientAllowList(remote)):
		log.Debugf("Setting client permissions")
		a.setClientPermissions(user, caller, clientClaims, log)

	// Else in the case where an allow list is configured we set server permissions on other conns
	case setServerPerms || len(a.clientAllowList) > 0:
		a.setServerPermissions(user, serverClaims, log)

	case pipeConnection:
		log.Debugf("Allowing pipe connection without any limitations")
	}

	if user.Account != nil {
		log.Debugf("Registering user '%s' in account '%s'", user.Username, user.Account.Name)
	} else {
		log.Debugf("Registering user '%s' in default account", user.Username)
	}

	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) handleUnverifiedSystemAccount(c server.ClientAuthentication, conn *tls.ConnectionState, log *logrus.Entry) (bool, error) {
	if conn == nil {
		return false, fmt.Errorf("requires TLS")
	}

	remote := c.RemoteAddress()
	opts := c.GetOpts()
	jwts := opts.Token

	if jwts == emptyString {
		return false, fmt.Errorf("no JWT token received")
	}

	purpose := tokens.TokenPurpose(jwts)
	log = log.WithFields(logrus.Fields{"jwt_auth": true, "purpose": purpose})
	log.Infof("Performing JWT based authentication verification for system account access")

	if purpose != tokens.ClientIDPurpose {
		return false, fmt.Errorf("client token required")
	}

	if c.Kind() != server.CLIENT {
		return false, fmt.Errorf("a client JWT was presented by a %d connection", c.Kind())
	}

	claims, err := a.parseClientIDJWT(jwts)
	if err != nil {
		log.Errorf("could not parse JWT from %s: %s", remote.String(), err)
		return false, fmt.Errorf("invalid JWT token")
	}

	if claims.Permissions == nil || !claims.Permissions.SystemUser {
		return false, fmt.Errorf("no system_user claim")
	}

	nonce := c.GetNonce()
	_, err = a.verifyNonceSignature(nonce, opts.Sig, claims.PublicKey, log)
	if err != nil {
		log.Errorf("nonce signature verification failed: %s", err)
		return false, fmt.Errorf("invalid nonce signature")
	}

	return a.handleVerifiedSystemAccount(c, log)
}

func (a *ChoriaAuth) handleVerifiedSystemAccount(c server.ClientAuthentication, log *logrus.Entry) (bool, error) {
	if a.systemUser == "" {
		return false, fmt.Errorf("system user is required")
	}

	if a.systemPass == "" {
		return false, fmt.Errorf("system password is required")
	}

	if a.systemAccount == nil {
		return false, fmt.Errorf("system account is not set")
	}

	opts := c.GetOpts()

	if !(opts.Username == a.systemUser && opts.Password == a.systemPass) {
		return false, fmt.Errorf("invalid system credentials")
	}

	user := a.createUser(c)
	user.Account = a.systemAccount

	log.Debugf("Registering user '%s' in account '%s'", user.Username, user.Account.Name)
	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) handleProvisioningUserConnection(c server.ClientAuthentication, tlsVerified bool) (bool, error) {
	if !tlsVerified {
		return false, fmt.Errorf("provisioning user is only allowed over verified TLS connections")
	}

	if a.provPass == emptyString {
		return false, fmt.Errorf("provisioning user password not enabled")
	}

	if a.provisioningAccount == nil {
		return false, fmt.Errorf("provisioning account is not set")
	}

	if !a.isTLS {
		return false, fmt.Errorf("provisioning user access requires TLS")
	}

	if c.GetTLSConnectionState() == nil {
		return false, fmt.Errorf("provisioning user can only connect over tls")
	}

	opts := c.GetOpts()

	if opts.Password == emptyString {
		return false, fmt.Errorf("password required")
	}

	if a.provPass != opts.Password {
		return false, fmt.Errorf("invalid provisioner password supplied")
	}

	user := a.createUser(c)
	user.Account = a.provisioningAccount

	a.log.Debugf("Registering user '%s' in account '%s'", user.Username, user.Account.Name)
	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) handleUnverifiedProvisioningConnection(c server.ClientAuthentication) (bool, error) {
	if a.provisioningTokenSigner == emptyString {
		return false, fmt.Errorf("provisioning is not enabled")
	}

	if !util.FileExist(a.provisioningTokenSigner) {
		return false, fmt.Errorf("provisioning signer certificate %s does not exist", a.provisioningTokenSigner)
	}

	if a.provisioningAccount == nil {
		return false, fmt.Errorf("provisioning account is not set")
	}

	opts := c.GetOpts()
	if opts.Username == provisioningUser {
		return false, fmt.Errorf("provisioning user requires verified TLS")
	}

	user := a.createUser(c)
	user.Account = a.provisioningAccount

	switch {
	case opts.Token != emptyString:
		_, err := tokens.ParseProvisioningTokenWithKeyfile(opts.Token, a.provisioningTokenSigner)
		if err != nil {
			return false, err
		}

		a.log.Debugf("Allowing a provisioning server from using unverified TLS connection from %s", c.RemoteAddress().String())

	default:
		return false, fmt.Errorf("provisioning requires a token")
	}

	// anything that get this far has to be a server and so we unconditionally set server
	// only permissions, and only to agents provisioning would bother hosting.
	//
	// We also allow provisioning.registration.> to allow a mode where prov mode servers
	// would be publishing some known metadata, by convention, this is the only place they
	// can publish to
	user.Permissions.Subscribe = &server.SubjectPermission{
		Allow: []string{
			"provisioning.node.>",
			"provisioning.broadcast.agent.discovery",
			"provisioning.broadcast.agent.rpcutil",
			"provisioning.broadcast.agent.choria_util",
			"provisioning.broadcast.agent.choria_provision",
		},
	}

	user.Permissions.Publish = &server.SubjectPermission{
		Allow: []string{
			"choria.lifecycle.>",
			"provisioning.reply.>",
			"provisioning.registration.>",
		},
	}

	a.log.Debugf("Registering user '%s' in account '%s'", user.Username, user.Account.Name)
	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) parseServerJWT(jwts string) (claims *tokens.ServerClaims, err error) {
	if a.serverJwtSigner == emptyString {
		return nil, fmt.Errorf("JWT Signer not set in plugin.choria.network.server_signer_cert, denying all clients")
	}

	if jwts == emptyString {
		return nil, fmt.Errorf("no JWT received")
	}

	claims, err = tokens.ParseServerTokenWithKeyfile(jwts, a.serverJwtSigner)
	if err != nil {
		return nil, err
	}

	if claims.ChoriaIdentity == emptyString {
		return nil, fmt.Errorf("identity not in claims")
	}

	if claims.PublicKey == emptyString {
		return nil, fmt.Errorf("no public key in claims")
	}

	return claims, nil
}

func (a *ChoriaAuth) parseClientIDJWT(jwts string) (claims *tokens.ClientIDClaims, err error) {
	if a.clientJwtSigner == emptyString {
		return nil, fmt.Errorf("JWT Signer not set in plugin.choria.network.client_signer_cert, denying all clients")
	}

	if jwts == emptyString {
		return nil, fmt.Errorf("no JWT received")
	}

	claims, err = tokens.ParseClientIDTokenWithKeyfile(jwts, a.clientJwtSigner, true)
	if err != nil {
		return nil, err
	}

	if claims.CallerID == emptyString {
		return nil, fmt.Errorf("no callerid in claims")
	}

	if claims.PublicKey == emptyString {
		return nil, fmt.Errorf("no public key in claims")
	}

	return claims, nil
}

func (a *ChoriaAuth) setMinimalClientPermissions(_ *server.User, caller string, subs []string, pubs []string) ([]string, []string) {
	replys := "*.reply.>"
	if caller != emptyString {
		replys = fmt.Sprintf("*.reply.%x.>", md5.Sum([]byte(caller)))
		a.log.Debugf("Creating ACLs for a private reply subject on %s", replys)
	}

	subs = append(subs, replys)
	pubs = append(pubs,
		"*.broadcast.agent.>",
		"*.broadcast.service.>",
		"*.node.>",
		"choria.federation.*.federation",
	)

	return subs, pubs
}

func (a *ChoriaAuth) setStreamsAdminPermissions(user *server.User, subs []string, pubs []string) ([]string, []string) {
	if user.Account != a.choriaAccount {
		return subs, pubs
	}

	subs = append(subs, "$JS.EVENT.>")
	pubs = append(pubs, "$JS.>")

	return subs, pubs
}

func (a *ChoriaAuth) setStreamsUserPermissions(user *server.User, subs []string, pubs []string) ([]string, []string) {
	if user.Account != a.choriaAccount {
		return subs, pubs
	}

	pubs = append(pubs,
		"$JS.API.INFO",
		"$JS.API.STREAM.NAMES",
		"$JS.API.STREAM.LIST",
		"$JS.API.STREAM.INFO.*",
		"$JS.API.STREAM.MSG.GET.*",
		"$JS.API.STREAM.MSG.DELETE.*",
		"$JS.API.DIRECT.GET.*",
		"$JS.API.DIRECT.GET.*.>",
		"$JS.API.CONSUMER.CREATE.*",
		"$JS.API.CONSUMER.CREATE.*.>",
		"$JS.API.CONSUMER.DURABLE.CREATE.*.*",
		"$JS.API.CONSUMER.DELETE.*.*",
		"$JS.API.CONSUMER.NAMES.*",
		"$JS.API.CONSUMER.LIST.*",
		"$JS.API.CONSUMER.INFO.*.*",
		"$JS.API.CONSUMER.MSG.NEXT.*.*",
		"$JS.ACK.>",
		"$JS.FC.>")

	return subs, pubs
}

func (a *ChoriaAuth) setEventsViewerPermissions(user *server.User, subs []string, pubs []string) ([]string, []string) {
	switch user.Account {
	case a.choriaAccount:
		subs = append(subs,
			"choria.lifecycle.event.>",
			"choria.machine.watcher.>",
			"choria.machine.transition")
	case a.provisioningAccount:
		// provisioner should only listen to one specific kind of event, not strictly needed but its what it is
		subs = append(subs, "choria.lifecycle.event.*.provision_mode_server")
	}

	return subs, pubs
}

func (a *ChoriaAuth) setClientGovernorPermissions(user *server.User, subs []string, pubs []string) ([]string, []string) {
	if user.Account != a.choriaAccount {
		return subs, pubs
	}

	pubs = append(pubs, "*.governor.*")

	return subs, pubs
}

func (a *ChoriaAuth) setElectionPermissions(user *server.User, subs []string, pubs []string) ([]string, []string) {
	switch user.Account {
	case a.choriaAccount:
		pubs = append(pubs,
			"$JS.API.STREAM.INFO.KV_CHORIA_LEADER_ELECTION",
			"$KV.CHORIA_LEADER_ELECTION.>")
	case a.provisioningAccount:
		// provisioner account is special and can only access one very specific election
		pubs = append(pubs,
			"choria.streams.STREAM.INFO.KV_CHORIA_LEADER_ELECTION",
			"$KV.CHORIA_LEADER_ELECTION.provisioner")
	}

	return subs, pubs
}

func (a *ChoriaAuth) setClientTokenPermissions(user *server.User, caller string, perms *tokens.ClientPermissions, log *logrus.Entry) (pubs []string, subs []string, err error) {
	if perms != nil && perms.OrgAdmin {
		log.Infof("Granting user access to all subjects (OrgAdmin)")
		return allSubjects, allSubjects, nil
	}

	subs, pubs = a.setMinimalClientPermissions(user, caller, subs, pubs)

	if perms == nil {
		return pubs, subs, nil
	}

	// Can access full Streams Features
	if perms.StreamsAdmin {
		log.Infof("Granting user Streams Admin access")
		subs, pubs = a.setStreamsAdminPermissions(user, subs, pubs)
	}

	// Can use streams but not make new ones etc
	if perms.StreamsUser {
		log.Infof("Granting user Streams User access")
		subs, pubs = a.setStreamsUserPermissions(user, subs, pubs)
	}

	// Lifecycle and auto agent events
	if perms.EventsViewer {
		log.Infof("Granting user Events Viewer access")
		subs, pubs = a.setEventsViewerPermissions(user, subs, pubs)
	}

	// KV based elections
	if perms.ElectionUser {
		log.Infof("Granting user Leader Election access")
		subs, pubs = a.setElectionPermissions(user, subs, pubs)
	}

	if perms.Governor && (perms.StreamsUser || perms.StreamsAdmin) {
		log.Infof("Granting user Governor access")
		subs, pubs = a.setClientGovernorPermissions(user, subs, pubs)
	}

	return pubs, subs, nil
}

func (a *ChoriaAuth) setClientPermissions(user *server.User, caller string, client *tokens.ClientIDClaims, log *logrus.Entry) {
	user.Permissions.Subscribe = &server.SubjectPermission{}
	user.Permissions.Publish = &server.SubjectPermission{}

	var perms *tokens.ClientPermissions
	if client != nil {
		perms = client.Permissions
	}

	pubs, subs, err := a.setClientTokenPermissions(user, caller, perms, log)
	if err != nil {
		log.Warnf("Could not determine permissions for user, denying all: %s", err)
		user.Permissions.Subscribe.Deny = allSubjects
		user.Permissions.Publish.Deny = allSubjects
	} else {
		user.Permissions.Subscribe.Allow = subs
		user.Permissions.Publish.Allow = pubs
	}

	log.Debugf("Setting sub permissions: %#v", user.Permissions.Subscribe)
	log.Debugf("Setting pub permissions: %#v", user.Permissions.Publish)
	if user.Permissions.Response != nil {
		log.Debugf("Setting resp permissions: %#v", user.Permissions.Response)
	}
}

func (a *ChoriaAuth) setDenyServersPermissions(user *server.User) {
	user.Permissions.Subscribe = &server.SubjectPermission{
		Deny: allSubjects,
	}

	user.Permissions.Publish = &server.SubjectPermission{
		Deny: allSubjects,
	}
}

func (a *ChoriaAuth) setClaimsBasedServerPermissions(user *server.User, claims *tokens.ServerClaims, log *logrus.Entry) {
	if len(claims.Collectives) == 0 {
		log.Warnf("no collectives in server token, denying access")
		a.setDenyServersPermissions(user)
		return
	}

	user.Permissions.Subscribe = &server.SubjectPermission{}
	user.Permissions.Publish = &server.SubjectPermission{
		Allow: []string{
			"choria.lifecycle.>",
			"choria.machine.transition",
			"choria.machine.watcher.>",
		},
	}

	user.Permissions.Publish.Allow = append(user.Permissions.Publish.Allow, claims.AdditionalPublishSubjects...)

	for _, c := range claims.Collectives {
		user.Permissions.Publish.Allow = append(user.Permissions.Publish.Allow,
			fmt.Sprintf("%s.reply.>", c),
			fmt.Sprintf("%s.broadcast.agent.registration", c),
			fmt.Sprintf("choria.federation.%s.collective", c),
		)

		user.Permissions.Subscribe.Allow = append(user.Permissions.Subscribe.Allow,
			fmt.Sprintf("%s.broadcast.agent.>", c),
			fmt.Sprintf("%s.node.%s", c, claims.ChoriaIdentity),
			fmt.Sprintf("%s.reply.%x.>", c, md5.Sum([]byte(claims.ChoriaIdentity))),
		)

		if claims.Permissions != nil {
			if claims.Permissions.ServiceHost {
				user.Permissions.Subscribe.Allow = append(user.Permissions.Subscribe.Allow,
					fmt.Sprintf("%s.broadcast.service.>", c),
				)
			}

			if claims.Permissions.Submission {
				user.Permissions.Publish.Allow = append(user.Permissions.Publish.Allow,
					fmt.Sprintf("%s.submission.in.>", c),
				)
			}

			if claims.Permissions.Governor && claims.Permissions.Streams {
				user.Permissions.Publish.Allow = append(user.Permissions.Publish.Allow,
					fmt.Sprintf("%s.governor.*", c),
				)
			}
		}
	}

	if claims.Permissions != nil && claims.Permissions.Streams {
		prefix := "$JS.API"
		if claims.OrganizationUnit != "choria" {
			prefix = "choria.streams"
		}

		user.Permissions.Publish.Allow = append(user.Permissions.Publish.Allow,
			fmt.Sprintf("%s.STREAM.INFO.*", prefix),
			fmt.Sprintf("%s.STREAM.MSG.GET.*", prefix),
			fmt.Sprintf("%s.STREAM.MSG.DELETE.*", prefix),
			fmt.Sprintf("%s.DIRECT.GET.*", prefix),
			fmt.Sprintf("%s.DIRECT.GET.*.>", prefix),
			fmt.Sprintf("%s.CONSUMER.CREATE.*", prefix),
			fmt.Sprintf("%s.CONSUMER.CREATE.*.>", prefix),
			fmt.Sprintf("%s.CONSUMER.DURABLE.CREATE.*.*", prefix),
			fmt.Sprintf("%s.CONSUMER.INFO.*.*", prefix),
			fmt.Sprintf("%s.CONSUMER.MSG.NEXT.*.*", prefix),
			"$JS.ACK.>",
			"$JS.FC.>",
		)
	}
}

func (a *ChoriaAuth) setDefaultServerPermissions(user *server.User) {
	user.Permissions.Subscribe = &server.SubjectPermission{
		Deny: []string{
			"*.reply.>",
			"choria.federation.>",
			"choria.lifecycle.>",
		},
	}

	user.Permissions.Publish = &server.SubjectPermission{
		Allow: allSubjects,

		Deny: []string{
			"*.broadcast.agent.>",
			"*.broadcast.service.>",
			"*.node.>",
			"choria.federation.*.federation",
		},
	}
}

func (a *ChoriaAuth) setServerPermissions(user *server.User, claims *tokens.ServerClaims, log *logrus.Entry) {
	switch {
	case a.denyServers:
		log.Debugf("Setting server permissions, denying servers")
		a.setDenyServersPermissions(user)

	case claims != nil:
		log.Debugf("Setting server permissions based on token claims")
		a.setClaimsBasedServerPermissions(user, claims, log)

	default:
		log.Debugf("Setting default server permissions")
		a.setDefaultServerPermissions(user)
	}
}

func (a *ChoriaAuth) remoteInClientAllowList(remote net.Addr) bool {
	if len(a.clientAllowList) == 0 {
		return true
	}

	if remote == nil {
		return false
	}

	host, _, err := net.SplitHostPort(remote.String())
	if err != nil {
		a.log.Warnf("Could not extract host from remote, not allowing access to client targets: '%s': %s", remote.String(), err)

		return false
	}

	for _, allowed := range a.clientAllowList {
		if host == allowed {
			return true
		}

		if strings.Contains(allowed, "/") {
			_, ipnet, err := net.ParseCIDR(allowed)
			if err != nil {
				a.log.Warnf("Could not parse %s as a cidr: %s", allowed, err)
				continue
			}

			if ipnet.Contains(net.ParseIP(host)) {
				return true
			}
		}
	}

	return false
}

func (a *ChoriaAuth) isProvisionUser(c server.ClientAuthentication) bool {
	opts := c.GetOpts()
	return opts.Username == provisioningUser
}

func (a *ChoriaAuth) isSystemUser(c server.ClientAuthentication) bool {
	if a.systemUser == "" {
		return false
	}

	opts := c.GetOpts()
	return opts.Username == a.systemUser
}

func (a *ChoriaAuth) createUser(c server.ClientAuthentication) *server.User {
	opts := c.GetOpts()

	acct := a.choriaAccount

	return &server.User{
		Username:    opts.Username,
		Password:    opts.Password,
		Account:     acct,
		Permissions: &server.Permissions{},
	}
}
