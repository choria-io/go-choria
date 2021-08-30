package inter

import (
	"github.com/choria-io/go-choria/config"
)

// ConfigurationProvider provides runtime Choria configuration
type ConfigurationProvider interface {
	Configuration() *config.Config
}
