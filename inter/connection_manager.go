package inter

import (
	"context"

	"github.com/choria-io/go-choria/srvcache"
	log "github.com/sirupsen/logrus"
)

// ConnectionManager is capable of being a factory for connection, mcollective.Choria is one
type ConnectionManager interface {
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (conn Connector, err error)
}
