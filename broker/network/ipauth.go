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

	"github.com/choria-io/go-choria/choria"
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
	allowList         []string
	anonTLS           bool
	denyServers       bool
	userJWTSignerCert string
	provJWTSignerCert string
	log               *logrus.Entry
	srv               *Server
}

// Check checks and registers the incoming connection
func (a *IPAuth) Check(c server.ClientAuthentication) (verified bool) {
	var (
		err      error
		account  string
		caller   string
		user     = a.createUser(c)
		remote   = c.RemoteAddress()
		jwts     = c.GetOpts().Token
		connType = choria.UnknownJWT
	)

	if remote == nil {
		a.log.Warn("Denying unknown remote connection")
		return false
	}

	if a.anonTLS && jwts == "" {
		a.log.Warnf("no JWT received from %s while in anonymous mode, denying connection", remote.String())
		return false
	}

	connType, err = a.detectConnType(jwts, remote)
	if err != nil {
		a.log.Warnf("Could not detect connection type for %s, denying connection: %s", remote.String(), err)
		return false
	}

	// only if allow lists are set else its a noop and all traffic is passed
	switch connType {
	case choria.ClientJWT:
		a.log.Debugf("Treating %s as a client", remote.String())

		a.setClientPermissions(user, caller)
		if jwts != "" {
			caller, account, err = a.parseUserJWT(jwts)
			if err != nil {
				a.log.Warnf("Could not parse JWT from %s, denying client: %s", remote.String(), err)
				return false
			}
		}

	case choria.ServerJWT:
		a.log.Debugf("Treating %s as a server", remote.String())

		a.setServerPermissions(user)
		if jwts != "" {
			account, err = a.parseProvisionJWT(jwts)
			if err != nil {
				a.log.Warnf("Could not parse JWT from %s, denying server: %s", remote.String(), err)
				return false
			}
		}

	case choria.UnknownJWT:
		a.log.Debugf("Treating %s as an unknown connection type", remote.String())
	}

	err = a.addUSerToAccount(user, account, remote)
	if err != nil {
		a.log.Warnf("Could not add user %s to account %s, denying user: %s", remote.String(), account, err)
		return false
	}

	c.RegisterUser(user)

	return true
}

func (a *IPAuth) detectConnType(jwts string, remote net.Addr) (choria.JWTType, error) {
	var connType choria.JWTType
	var err error

	if jwts != "" {
		a.log.Debugf("Using JWT from %s to detect connection type", remote.String())

		connType, err = a.detectJWTType(jwts)
		if err != nil {
			return "", fmt.Errorf("invalid JWT")
		}
	}

	if connType == choria.UnknownJWT {
		switch {
		case a.remoteInClientAllowList(remote):
			connType = choria.ClientJWT
		case len(a.allowList) > 0:
			connType = choria.ServerJWT
		}
	}

	return connType, nil
}

func (a *IPAuth) addUSerToAccount(user *server.User, account string, remote net.Addr) error {
	if account == "" {
		return nil
	}

	a.srv.mu.Lock()
	srv := a.srv.gnatsd
	a.srv.mu.Unlock()

	if srv == nil {
		return fmt.Errorf("no NATS Server instance found")
	}

	new := false

	a.log.Debugf("Joining connection %s to account %s", remote.String(), account)
	user.Account, new = srv.LookupOrRegisterAccount(account)
	if new {
		a.log.Infof("Created new account %s", account)
	}

	// TODO only to the shared admin layer - once it exist
	user.Account.AddStreamExport("choria.lifecycle.event.>", nil)
	user.Account.AddStreamExport("choria.machine.>", nil)

	if user.Account.IsExpired() {
		a.log.Warnf("Account %s is expired", user.Account.Name)
	}

	return nil
}

func (a *IPAuth) detectJWTType(jwts string) (choria.JWTType, error) {
	if jwts == "" {
		return "", fmt.Errorf("no JWT provided")
	}

	claims := jwt.MapClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(jwts, &claims)
	if err != nil {
		return "", err
	}

	subj, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("no subject found in claims")
	}

	iss, ok := claims["iss"].(string)
	// TODO remove this after AAA update
	if iss == "Choria Userlist Authenticator" {
		return choria.ClientJWT, nil
	}

	switch subj {
	case choria.ServerJWT:
		return choria.ServerJWT, nil
	case choria.ClientJWT:
		return choria.ClientJWT, nil
	}

	return "", fmt.Errorf("unknown token")
}

func (a *IPAuth) parseProvisionJWT(jwts string) (string, error) {
	if a.provJWTSignerCert == "" {
		return "", fmt.Errorf("provisioning signer key not set")
	}

	_, claims, err := a.parseJWT(jwts, a.provJWTSignerCert)
	if err != nil {
		return "", err
	}

	subj, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("no sub in claims")
	}

	if subj != choria.ServerJWT {
		return "", fmt.Errorf("token is not a server JWT")
	}

	org, ok := claims["chou"].(string)

	return org, nil
}

func (a *IPAuth) parseJWT(jwts string, pk string) (*jwt.Token, jwt.MapClaims, error) {
	if pk == "" {
		return nil, nil, fmt.Errorf("no public certificate supplied")
	}

	signKey, err := a.loadKey(pk)
	if err != nil {
		return nil, nil, fmt.Errorf("signing key error: %s", err)
	}

	token, err := jwt.Parse(jwts, func(token *jwt.Token) (interface{}, error) {
		return signKey, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("invalid JWT: %s", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, fmt.Errorf("invalid claims")
	}

	err = claims.Valid()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid claims")
	}

	return token, claims, nil
}

func (a *IPAuth) parseUserJWT(jwts string) (string, string, error) {
	if a.userJWTSignerCert == "" {
		return "", "", fmt.Errorf("anonymous TLS JWT Signer not set in plugin.choria.security.request_signing_certificate, denying all clients")
	}

	_, claims, err := a.parseJWT(jwts, a.userJWTSignerCert)
	if err != nil {
		return "", "", err
	}

	subj, ok := claims["sub"].(string)
	if !ok {
		return "", "", fmt.Errorf("no sub in claims")
	}

	iss, ok := claims["iss"].(string)
	// TODO remove this after AAA update
	if !(subj == choria.ClientJWT || iss == "Choria Userlist Authenticator") {
		return "", "", fmt.Errorf("token is not a client JWT")
	}

	caller, ok := claims["callerid"].(string)
	if !ok {
		return "", "", fmt.Errorf("no callerid in claims")
	}

	if caller == "" {
		return "", "", fmt.Errorf("empty callerid in claims")
	}

	org, ok := claims["chou"].(string)

	// TODO remove after aaa update
	if caller == "up=rip" {
		org = "development"
	}

	return caller, org, nil
}

func (a *IPAuth) loadKey(path string) (*rsa.PublicKey, error) {
	certBytes, err := ioutil.ReadFile(path)
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
