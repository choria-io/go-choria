package federation

import (
	"sync"
	"time"

	"github.com/choria-io/go-choria/mcollective"
)

type FederationBroker struct {
	Stats   *Stats
	statsMu sync.Mutex

	clusterName  string
	instanceName string
}

func NewFederationBroker(clusterName string, instanceName string, choria *mcollective.Choria) (broker *FederationBroker, err error) {
	broker = &FederationBroker{
		clusterName:  clusterName,
		instanceName: instanceName,
		Stats: &Stats{
			ConfigFile:      &choria.Config.ConfigFile,
			StartTime:       time.Now(),
			Status:          "unknown",
			CollectiveStats: &WorkerStats{ConnectedServer: "unknown"},
			FederationStats: &WorkerStats{ConnectedServer: "unknown"},
		},
	}

	return
}
