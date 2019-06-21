package network

import (
	"net"
	"strings"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/sirupsen/logrus"
)

// IPAuth implements gnatsd server.Authentication interface and
// allows IP limits to be configured, connections that do not match
// the configured IP or CIDRs are not allowed to publish to the
// network targets used by clients to request actions on nodes
type IPAuth struct {
	allowList []string
	log       *logrus.Entry
}

// Check checks and registers the incoming connection
func (a *IPAuth) Check(c server.ClientAuthentication) (verified bool) {
	user := a.createUser(c)

	remote := c.RemoteAddress()
	if remote != nil && !a.remoteInClientAllowList(c.RemoteAddress()) {
		a.setServerPermissions(user)
	}

	c.RegisterUser(user)

	return true
}

func (a *IPAuth) setServerPermissions(user *server.User) {
	user.Permissions.Subscribe = &server.SubjectPermission{
		Deny: []string{
			"*.reply.>",
			"choria.federation.>",
			"choria.lifecycle.>",
		},
	}

	user.Permissions.Publish = &server.SubjectPermission{
		Allow: []string{
			"*.broadcast.agent.registration",
		},

		Deny: []string{
			"*.broadcast.agent.>",
			"*.node.>",
			"choria.federation.*.federation",
		},
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
