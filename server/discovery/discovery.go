package discovery

import (
	"github.com/choria-io/go-choria/server/discovery/agents"
	"github.com/choria-io/go-choria/server/discovery/classes"
	"github.com/choria-io/go-choria/server/discovery/facts"
	"github.com/choria-io/go-choria/server/discovery/identity"
	"github.com/choria-io/go-config"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/sirupsen/logrus"
)

// Manager manages the full discovery life cycle
type Manager struct {
	fw  *choria.Framework
	cfg *config.Config
	log *logrus.Entry
}

// New creates a new discovery Manager
func New(framework *choria.Framework, logger *logrus.Entry) *Manager {
	return &Manager{
		fw:  framework,
		cfg: framework.Configuration(),
		log: logger.WithFields(logrus.Fields{"subsystem": "discovery"}),
	}
}

// ShouldProcess checks all filters against methods for filtering
// and returns boolean if it matches
func (mgr *Manager) ShouldProcess(request protocol.Request, knownAgents []string) bool {
	filter, _ := request.Filter()
	passed := 0

	if filter.Empty() {
		mgr.log.Debugf("Matching request %s with empty filter", request.RequestID())
		passed++
	}

	if len(filter.ClassFilters()) > 0 {
		if classes.Match(filter.ClassFilters(), mgr.cfg.ClassesFile, mgr.log) {
			mgr.log.Debugf("Matching request %s with class filters '%#v'", request.RequestID(), filter.ClassFilters())
			passed++
		} else {
			mgr.log.Debugf("Not matching request %s with class filters '%#v'", request.RequestID(), filter.ClassFilters())
			return false
		}
	}

	if len(filter.AgentFilters()) > 0 {
		if agents.Match(filter.AgentFilters(), knownAgents) {
			mgr.log.Debugf("Matching request %s with agent filters '%#v'", request.RequestID(), filter.AgentFilters())
			passed++
		} else {
			mgr.log.Debugf("Not matching request %s with agent filters '%#v'", request.RequestID(), filter.AgentFilters())
			return false
		}
	}

	if len(filter.IdentityFilters()) > 0 {
		if identity.Match(filter.IdentityFilters(), mgr.cfg.Identity) {
			mgr.log.Debugf("Matching request %s with identity filters '%#v'", request.RequestID(), filter.IdentityFilters())
			passed++
		} else {
			mgr.log.Debugf("Not matching request %s with identity filters '%#v'", request.RequestID(), filter.IdentityFilters())
			return false
		}
	}

	if len(filter.FactFilters()) > 0 {
		if facts.Match(filter.FactFilters(), mgr.fw, mgr.log) {
			mgr.log.Debugf("Matching request %s based on fact filters '%#v'", request.RequestID(), filter.FactFilters())
			passed++
		} else {
			mgr.log.Debugf("Not matching request %s based on fact filters '%#v'", request.RequestID(), filter.FactFilters())
			return false
		}
	}

	if len(filter.CompoundFilters()) > 0 {
		mgr.log.Warnf("Compound filters are not supported, not matching request %s with filter '%#v'", request.RequestID(), filter.CompoundFilters())
		return false
	}

	return passed > 0
}
