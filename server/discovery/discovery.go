package discovery

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/data/ddl"
)

type ServerInfoSource interface {
	Classes() []string
	Facts() json.RawMessage
	Identity() string
	KnownAgents() []string
	DataFuncMap() (ddl.FuncMap, error)
}

// Manager manages the full discovery life cycle
type Manager struct {
	cfg *config.Config
	si  ServerInfoSource
	log *logrus.Entry
}

// New creates a new discovery Manager
func New(cfg *config.Config, si ServerInfoSource, logger *logrus.Entry) *Manager {
	return &Manager{
		cfg: cfg,
		si:  si,
		log: logger.WithFields(logrus.Fields{"subsystem": "discovery"}),
	}
}

// ShouldProcess checks all filters against methods for filtering and returns boolean if it matches
func (mgr *Manager) ShouldProcess(request protocol.Request) bool {
	filter, _ := request.Filter()

	return filter.MatchServerRequest(request, mgr.si, mgr.log.WithField("request", request.RequestID()))
}
