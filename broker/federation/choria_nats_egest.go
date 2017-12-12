package federation

import (
	"context"
	"fmt"
	"strings"

	"github.com/choria-io/go-choria/broker/federation/stats"
	"github.com/choria-io/go-choria/choria"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func NewChoriaNatsEgest(workers int, mode int, capacity int, broker *FederationBroker, logger *log.Entry) (*pooledWorker, error) {
	worker, err := PooledWorkerFactory("choria_nats_egest", workers, mode, capacity, broker, logger, func(ctx context.Context, self *pooledWorker, i int, logger *log.Entry) {
		defer self.wg.Done()

		var nc choria.Connector
		var err error

		nc, err = self.connection.NewConnector(ctx, self.servers, self.Name(), logger)
		if err != nil {
			logger.Errorf("Could not start NATS connection for worker %d: %s", i, err.Error())
			return
		}

		workeri := fmt.Sprintf("%d", i)
		rctr := stats.ReceivedMsgsCtr.WithLabelValues("nats_egest", workeri, "")
		pctr := stats.PublishedMsgsCtr.WithLabelValues("nats_egest", workeri, "")
		ectr := stats.ErrorCtr.WithLabelValues("nats_egest", workeri, "")
		timer := stats.ProcessTime.WithLabelValues("nats_egest", workeri, "")

		handler := func(cm chainmessage) {
			obs := prometheus.NewTimer(timer)
			defer obs.ObserveDuration()

			if len(cm.Targets) == 0 {
				logger.Errorf("Received message '%s' with no targets, discarding: %#v", cm.RequestID, cm)
				ectr.Inc()
				return
			}

			rctr.Inc()

			logger.Debugf("Publishing message '%s' to %d target(s)", cm.RequestID, len(cm.Targets))

			cm.Seen = append(cm.Seen, fmt.Sprintf("%s:%d", self.Name(), i))
			cm.Seen = append(cm.Seen, nc.ConnectedServer())

			if len(cm.Seen) >= 3 {
				mid := fmt.Sprintf("%s (%s)", self.choria.Config.Identity, strings.Join(cm.Seen[1:len(cm.Seen)-1], ", "))
				cm.Message.RecordNetworkHop(cm.Seen[0], mid, cm.Seen[len(cm.Seen)-1])
			}

			j, err := cm.Message.JSON()
			if err != nil {
				logger.Errorf("Could not JSON encode message '%s': %s", cm.RequestID, err.Error())
				ectr.Inc()
				return
			}

			for _, target := range cm.Targets {
				if err = nc.PublishRaw(target, []byte(j)); err != nil {
					logger.Errorf("Could not publish message '%s' to '%s': %s", cm.RequestID, target, err.Error())
					ectr.Inc()
					continue
				}
				pctr.Inc()
			}
		}

		for {
			var cm chainmessage

			select {
			case cm = <-self.in:
				handler(cm)
			case <-ctx.Done():
				logger.Infof("Worker routine %s exiting", self.Name())
				return
			}
		}
	})

	return worker, err
}
