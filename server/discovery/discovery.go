package discovery

import (
	"github.com/choria-io/go-config"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/protocol"
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

	return filter.MatchRequest(request, knownAgents, mgr.cfg.Identity, mgr.cfg.ClassesFile, mgr.cfg.FactSourceFile, mgr.log)
}
