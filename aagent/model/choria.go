package model

import (
	"encoding/json"

	"github.com/choria-io/go-choria/choria"
	"github.com/sirupsen/logrus"
)

// ChoriaProvider provides access to the choria framework
type ChoriaProvider interface {
	PublishRaw(string, []byte) error
	Logger(string) *logrus.Entry
	Identity() string
	PrometheusTextFileDir() string
	ScoutOverridesPath() string
	ServerStatusFile() (string, int)
	MainCollective() string
	Connector() choria.Connector
	Facts() json.RawMessage
}
