package network

import (
	"crypto/md5"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/providers/provtarget/builddefaults"
	"github.com/golang-jwt/jwt"
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
// AAA Service.
//
// Additionally when provisioning support is enabled any non TLS connection
// will be placed in the provisioning account and unable to connect to the
// fleet or provisioned nodes
type ChoriaAuth struct {
	allowList               []string
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
	case a.isTLS && !tlsVerified:
		verified, err = a.handleUnverifiedProvisioningConnection(c)
		if err != nil {
			a.log.Warnf("Handling unverified connection failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

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
		verified, err = a.handleSystemAccount(c)
		if err != nil {
			a.log.Warnf("Handling system user failed, denying %s: %s", c.RemoteAddress().String(), err)
		}

	default:
		if a.isTLS && !tlsVerified {
			a.log.Warnf("Rejecting non TLS client while in TLS mode")
			break
		}

		verified, err = a.handleDefaultConnection(c)
		if err != nil {
			a.log.Warnf("Handling normal connection failed, denying %s: %s", c.RemoteAddress().String(), err)
		}
	}

	// should be already but lets make sure
	if err != nil {
		verified = false
	}

	return verified
}

func (a *ChoriaAuth) handleDefaultConnection(c server.ClientAuthentication) (bool, error) {
	user := a.createUser(c)
	remote := c.RemoteAddress()
	opts := c.GetOpts()
	jwts := opts.Token
	caller := ""
	var err error

	if a.anonTLS {
		if remote == nil {
			return false, fmt.Errorf("unknown remote client while in AnonTLS mode")
		}

		caller, err = a.parseAnonTLSJWTUser(jwts)
		if err != nil {
			return false, fmt.Errorf("could not parse JWT from %s: %s", remote.String(), err)
		}
	}

	// only if allow lists are set else its a noop and all traffic is passed
	switch {
	case a.remoteInClientAllowList(remote):
		a.setClientPermissions(user, caller)

	case len(a.allowList) > 0:
		a.setServerPermissions(user)
	}

	if user.Account != nil {
		a.log.Debugf("Registering user %q in account %q", user.Username, user.Account.Name)
	} else {
		a.log.Debugf("Registering user %q in default account", user.Username)
	}

	c.RegisterUser(user)

	return true, nil
}

func (a *ChoriaAuth) handleSystemAccount(c server.ClientAuthentication) (bool, error) {
	if a.systemAccount == nil {
		return false, fmt.Errorf("system account is not set")
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
		cert, err := a.provisionerJWTSignerKey()
		if err != nil {
			return false, err
		}

		claims := &builddefaults.ProvClaims{}
		_, err = jwt.ParseWithClaims(opts.Token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unsupported signing method in token")
			}

			return cert, nil
		})
		if err != nil {
			return false, fmt.Errorf("could not parse provisioner token: %s", err)
		}

		if !claims.Secure {
			return false, fmt.Errorf("insecure provisioning client on TLS connection")
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

func (a *ChoriaAuth) parseAnonTLSJWTUser(jwts string) (string, error) {
	if a.jwtSigner == emptyString {
		return "", fmt.Errorf("anonymous TLS JWT Signer not set in plugin.choria.security.request_signing_certificate, denying all clients")
	}

	if jwts == emptyString {
		return "", fmt.Errorf("no JWT received")
	}

	signKey, err := a.clientJWTSignerKey()
	if err != nil {
		return "", fmt.Errorf("signing key error: %s", err)
	}

	token, err := jwt.Parse(jwts, func(token *jwt.Token) (interface{}, error) {
		return signKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid JWT: %s", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}

	err = claims.Valid()
	if err != nil {
		return "", fmt.Errorf("invalid claims")
	}

	caller, ok := claims["callerid"].(string)
	if !ok {
		return "", fmt.Errorf("no callerid in claims")
	}

	if caller == emptyString {
		return "", fmt.Errorf("empty callerid in claims")
	}

	return caller, nil
}

// reads plugin.choria.security.request_signing_certificate
func (a *ChoriaAuth) clientJWTSignerKey() (*rsa.PublicKey, error) {
	certBytes, err := os.ReadFile(a.jwtSigner)
	if err != nil {
		return nil, err
	}

	signKey, err := jwt.ParseRSAPublicKeyFromPEM(certBytes)
	if err != nil {
		return nil, err
	}

	return signKey, nil
}

// reads plugin.choria.network.provisioning.signer_cert
func (a *ChoriaAuth) provisionerJWTSignerKey() (*rsa.PublicKey, error) {
	certBytes, err := os.ReadFile(a.provisioningTokenSigner)
	if err != nil {
		return nil, err
	}

	signKey, err := jwt.ParseRSAPublicKeyFromPEM(certBytes)
	if err != nil {
		return nil, err
	}

	return signKey, nil
}

func (a *ChoriaAuth) setClientPermissions(user *server.User, caller string) {
	if !a.anonTLS {
		return
	}

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
	if len(a.allowList) == 0 {
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

	for _, allowed := range a.allowList {
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
	opts := c.GetOpts()
	return opts.Username != emptyString && opts.Password != emptyString && opts.Username == a.systemUser && opts.Password == a.systemPass
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
