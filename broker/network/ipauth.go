package network

import (
	"crypto/md5"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/sirupsen/logrus"
)

// IPAuth implements Nats Server server.Authentication interface and
// allows IP limits to be configured, connections that do not match
// the configured IP or CIDRs are not allowed to publish to the
// network targets used by clients to request actions on nodes.
//
// Additionally when the server is running in a mode where anonymous
// TLS connections is accepted then servers are entirely denied and
// clients are allowed but restricted based on the JWT issued by the
// AAA Service.
type IPAuth struct {
	allowList   []string
	anonTLS     bool
	denyServers bool
	jwtSigner   string
	log         *logrus.Entry
}

// Check checks and registers the incoming connection
func (a *IPAuth) Check(c server.ClientAuthentication) (verified bool) {
	user := a.createUser(c)
	remote := c.RemoteAddress()
	jwts := c.GetOpts().Token
	caller := ""

	var err error

	if a.anonTLS {
		if remote == nil {
			a.log.Warn("Denying unknown remote client while in AnonTLS mode")
			return false
		}

		caller, err = a.parseAnonTLSJWTUser(jwts)
		if err != nil {
			a.log.Warnf("Could not parse JWT from %s, denying client: %s", remote.String(), err)
			return false
		}
	}

	// only if allow lists are set else its a noop and all traffic is passed
	switch {
	case a.remoteInClientAllowList(remote):
		a.setClientPermissions(user, caller)

	case len(a.allowList) > 0:
		a.setServerPermissions(user)

	}

	c.RegisterUser(user)

	return true
}

func (a *IPAuth) parseAnonTLSJWTUser(jwts string) (string, error) {
	if a.jwtSigner == "" {
		return "", fmt.Errorf("anonymous TLS JWT Signer not set in plugin.choria.security.request_signing_certificate, denying all clients")
	}

	if jwts == "" {
		return "", fmt.Errorf("no JWT received")
	}

	signKey, err := a.jwtSignerKey()
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

	if caller == "" {
		return "", fmt.Errorf("empty callerid in claims")
	}

	return caller, nil
}

func (a *IPAuth) jwtSignerKey() (*rsa.PublicKey, error) {
	certBytes, err := ioutil.ReadFile(a.jwtSigner)
	if err != nil {
		return nil, err
	}

	signKey, err := jwt.ParseRSAPublicKeyFromPEM(certBytes)
	if err != nil {
		return nil, err
	}

	return signKey, nil
}

func (a *IPAuth) setClientPermissions(user *server.User, caller string) {
	if !a.anonTLS {
		return
	}

	replys := "*.reply.>"
	if caller != "" {
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
			"*.node.>",
			"choria.federation.*.federation",
		},
	}
}

func (a *IPAuth) setServerPermissions(user *server.User) {
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
				"*.node.>",
				"choria.federation.*.federation",
			},
		}
	}
}

func (a *IPAuth) remoteInClientAllowList(remote net.Addr) bool {
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

func (a *IPAuth) createUser(c server.ClientAuthentication) *server.User {
	opts := c.GetOpts()

	return &server.User{
		Username:    opts.Username,
		Password:    opts.Password,
		Permissions: &server.Permissions{},
	}
}
