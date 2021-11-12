// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/tokens"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/sirupsen/logrus"
)

// ChoriaAuth implements Nats Server server.Authentication interface and
// allows IP limits to be configured, connections that do not match
// the configured IP or CIDRs are not allowed to publish to the
// network targets used by clients to request actions on nodes.
//
// Additionally when the server is running in a mode where anonymous
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
	anonTLS                 bool
	isTLS                   bool
	denyServers             bool
	provisioningTokenSigner string
	jwtSigner               string
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

	switch {
	case a.isProvisionUser(c):
		if !tlsVerified {
			a.log.Warnf("Provision user is only allowed over verified TLS connections")
			return false
		}

		verified, err = a.handleProvisioningUserConnection(c)
		if err != nil {
			a.log.Warnf("Handling provisioning user connection failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

	case a.isSystemUser(c):
		if !tlsVerified {
			a.log.Warnf("System user is only allowed over verified TLS connections")
			return false
		}

		verified, err = a.handleSystemAccount(c)
		if err != nil {
			a.log.Warnf("Handling system user failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

	default:
		verified, err = a.handleDefaultConnection(c, tlsc, tlsVerified)
		if err != nil {
			a.log.Warnf("Handling normal connection failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

		if !verified && a.isTLS && !tlsVerified {
			verified, err = a.handleUnverifiedProvisioningConnection(c)
			if err != nil {
				a.log.Warnf("Handling unverified connection failed, denying %s: %s", c.RemoteAddress().String(), err)
			}
		}
	}

	// should be already but lets make sure
	if err != nil {
		verified = false
	}

	return verified
}

func (a *ChoriaAuth) handleDefaultConnection(c server.ClientAuthentication, conn *tls.ConnectionState, tlsVerified bool) (bool, error) {
	user := a.createUser(c)
	remote := c.RemoteAddress()
	opts := c.GetOpts()
	jwts := opts.Token
	caller := ""
	var err error

	log := a.log.WithField("mTLS", tlsVerified)
	if tlsVerified && len(conn.PeerCertificates) > 0 {
		log = log.WithField("subject", conn.PeerCertificates[0].Subject.CommonName)
	}
	if remote != nil {
		log = log.WithField("remote", remote.String())
	}

	// we only do JWT based auth in TLS mode
	if (a.anonTLS || jwts != "") && conn != nil {
		if remote == nil {
			return false, fmt.Errorf("remote client information is required in anonymous TLS or JWT signing modes")
		}

		caller, err = a.parseClientIDJWT(jwts)
		if err != nil {
			return false, fmt.Errorf("could not parse JWT from %s: %s", remote.String(), err)
		}
		user.Username = caller

		log = log.WithField("jwt_client", true)

		log.Debugf("Extracted caller id %s from JWT token", caller)
	}

	switch {
	// if a caller is set from the Client ID JWT we want to restrict it to just client stuff
	// if a client allow list is set and the client is in the ip range we restrict it also
	// else its default open like users with certs
	case (a.anonTLS || caller != "") && a.remoteInClientAllowList(remote):
		log = log.WithField("caller", caller)
		log.Debugf("Setting strict client permissions for %s", remote)
		a.setClientPermissions(user, caller)

	// Else in the case where an allow list is configured we set server permissions on other conns
	case len(a.clientAllowList) > 0:
		log.Debugf("Setting strict server permissions for %s", remote)
		a.setServerPermissions(user)
	}

	if user.Account != nil {
		log.Debugf("Registering user %q in account %q", user.Username, user.Account.Name)
	} else {
		log.Debugf("Registering user %q in default account", user.Username)
	}

	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) handleSystemAccount(c server.ClientAuthentication) (bool, error) {
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

	a.log.Debugf("Registering user %q in account %q", user.Username, user.Account.Name)
	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) handleProvisioningUserConnection(c server.ClientAuthentication) (bool, error) {
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

	a.log.Debugf("Registering user %q in account %q", user.Username, user.Account.Name)
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

	user := a.createUser(c)
	user.Account = a.provisioningAccount
	opts := c.GetOpts()

	if opts.Username == provisioningUser {
		return false, fmt.Errorf("provisioning user requires verified TLS")
	}

	switch {
	case opts.Token != emptyString:
		_, err := tokens.ParseProvisioningTokenWithKeyfile(opts.Token, a.provisioningTokenSigner)
		if err != nil {
			return false, err
		}

		a.log.Debugf("Allowing a provisioning server from %s using unverified TLS connection", c.RemoteAddress().String())

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

	a.log.Debugf("Registering user %s in account %s", user.Username, user.Account.Name)
	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) parseClientIDJWT(jwts string) (string, error) {
	if a.jwtSigner == emptyString {
		return "", fmt.Errorf("JWT Signer not set in plugin.choria.network.client_signer_cert, denying all clients")
	}

	if jwts == emptyString {
		return "", fmt.Errorf("no JWT received")
	}

	// Generally now we want to accept all mix mode clients who have a valid JWT, ie. one with the
	// correct purpose flag in addition to being valid, but to keep backwards compatibility with the
	// mode documented in https://choria.io/blog/post/2020/09/13/aaa_improvements/ we accept them in
	// the specific scenario where AnonTLS is configured without checking the purpose field
	claims, err := tokens.ParseClientIDTokenWithKeyfile(jwts, a.jwtSigner, !a.anonTLS)
	if err != nil {
		return "", err
	}

	if claims.CallerID == emptyString {
		return "", fmt.Errorf("no callerid in claims")
	}

	return claims.CallerID, nil
}

func (a *ChoriaAuth) setClientPermissions(user *server.User, caller string) {
	replys := "*.reply.>"
	if caller != emptyString {
		replys = fmt.Sprintf("*.reply.%x.>", md5.Sum([]byte(caller)))
		a.log.Debugf("Creating ACLs for a private reply subject on %s", replys)
	}

	user.Permissions.Subscribe = &server.SubjectPermission{
		Allow: []string{
			replys,
		},
	}

	user.Permissions.Publish = &server.SubjectPermission{
		Allow: []string{
			"*.broadcast.agent.>",
			"*.broadcast.service.>",
			"*.node.>",
			"choria.federation.*.federation",
		},
	}
}

func (a *ChoriaAuth) setServerPermissions(user *server.User) {
	matchAll := []string{">"}

	switch {
	case a.denyServers:
		user.Permissions.Subscribe = &server.SubjectPermission{
			Deny: matchAll,
		}

		user.Permissions.Publish = &server.SubjectPermission{
			Deny: matchAll,
		}

	default:
		user.Permissions.Subscribe = &server.SubjectPermission{
			Deny: []string{
				"*.reply.>",
				"choria.federation.>",
				"choria.lifecycle.>",
			},
		}

		user.Permissions.Publish = &server.SubjectPermission{
			Allow: matchAll,

			Deny: []string{
				"*.broadcast.agent.>",
				"*.broadcast.service.>",
				"*.node.>",
				"choria.federation.*.federation",
			},
		}
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

func (o *ChoriaAuth) isProvisionUser(c server.ClientAuthentication) bool {
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
