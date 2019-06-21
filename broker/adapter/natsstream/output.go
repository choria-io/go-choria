package natsstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/broker/adapter/stats"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
	uuid "github.com/gofrs/uuid"
	stan "github.com/nats-io/stan.go"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type stream struct {
	servers   func() ([]srvcache.Server, error)
	clusterID string
	clientID  string
	topic     string
	conn      stan.Conn
	log       *log.Entry
	name      string

	work chan adaptable
	quit chan bool
	mu   *sync.Mutex
}

type msg struct {
	Protocol  string    `json:"protocol"`
	Data      string    `json:"data"`
	Sender    string    `json:"sender"`
	Time      time.Time `json:"time"`
	RequestID string    `json:"requestid"`
}

func newStream(name string, work chan adaptable, logger *log.Entry) ([]*stream, error) {
	prefix := fmt.Sprintf("plugin.choria.adapter.%s.stream.", name)

	instances, err := strconv.Atoi(cfg.Option(prefix+"workers", "10"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", prefix+"workers")
	}

	servers := cfg.Option(prefix+"servers", "")
	if servers == "" {
		return nil, fmt.Errorf("No Stream servers configured, please set %s", prefix+"servers")
	}

	topic := cfg.Option(prefix+"topic", "")
	if topic == "" {
		topic = name
	}

	clusterID := cfg.Option(prefix+"clusterid", "")
	if clusterID == "" {
		return nil, fmt.Errorf("no ClusterID configured, please set %s", prefix+"clusterid'")
	}

	workers := []*stream{}

	for i := 0; i < instances; i++ {
		logger.Infof("Creating NATS Streaming Adapter %s NATS Streaming instance %d / %d publishing to %s on cluster %s", name, i, instances, topic, clusterID)

		wid, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("could not start output worker %d: %s", i, err)
		}

		iname := fmt.Sprintf("%s_%d-%s", name, i, strings.Replace(wid.String(), "-", "", -1))

		st := &stream{
			clusterID: clusterID,
			clientID:  iname,
			topic:     topic,
			name:      fmt.Sprintf("%s.%d", name, i),
			work:      work,
			log:       logger.WithFields(log.Fields{"side": "stream", "instance": i}),
			mu:        &sync.Mutex{},
		}
		st.servers = st.resolver(strings.Split(servers, ","))

		workers = append(workers, st)
	}

	return workers, nil
}

func (sc *stream) resolver(parts []string) func() ([]srvcache.Server, error) {
	servers, err := srvcache.StringHostsToServers(parts, "nats")
	return func() ([]srvcache.Server, error) {
		return servers, err
	}
}

func (sc *stream) connect(ctx context.Context, cm choria.ConnectionManager) error {
	if ctx.Err() != nil {
		return fmt.Errorf("Shutdown called")
	}

	reconn := make(chan struct{})

	nc, err := cm.NewConnector(ctx, sc.servers, sc.clientID, sc.log)
	if err != nil {
		return fmt.Errorf("Could not start NATS connection: %s", err)
	}

	start := func() error {
		sc.log.Infof("%s connecting to NATS Stream", sc.clientID)

		sc.mu.Lock()
		defer sc.mu.Unlock()

		ctr := 0

		for {
			ctr++

			if ctx.Err() != nil {
				return errors.New("shutdown called")
			}

			sc.conn, err = stan.Connect(sc.clusterID, sc.clientID, stan.NatsConn(nc.Nats()), stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
				sc.log.Errorf("NATS Streaming connection got disconnected, reconnecting: %s", reason)
				stats.ErrorCtr.WithLabelValues(sc.name, "output", cfg.Identity).Inc()
				reconn <- struct{}{}
			}))
			if err != nil {
				sc.log.Errorf("Could not create initial STAN connection, retrying: %s", err)
				backoff.FiveSec.InterruptableSleep(ctx, ctr)

				continue
			}

			break
		}

		return nil
	}

	watcher := func() {
		ctr := 0

		for {
			select {
			case <-reconn:
				ctr++

				sc.log.WithField("attempt", ctr).Infof("Attempting to reconnect NATS Stream after reconnection")

				backoff.FiveSec.InterruptableSleep(ctx, ctr)

				err := start()
				if err != nil {
					sc.log.Errorf("Could not restart NATS Streaming connection: %s", err)
					reconn <- struct{}{}
				}

			case <-ctx.Done():
				return
			}
		}
	}

	err = start()
	if err != nil {
		return fmt.Errorf("could not start initial NATS Streaming connection: %s", err)
	}

	go watcher()

	sc.log.Infof("%s connected to NATS Stream", sc.clientID)

	return nil
}

func (sc *stream) disconnect() {
	if sc.conn != nil {
		sc.log.Info("Disconnecting from NATS Streaming")
		sc.conn.Close()
	}
}

func (sc *stream) publisher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	bytes := stats.BytesCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	ectr := stats.ErrorCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	ctr := stats.ReceivedMsgsCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	timer := stats.ProcessTime.WithLabelValues(sc.name, "output", cfg.Identity)

	transformerf := func(r adaptable) {
		obs := prometheus.NewTimer(timer)
		defer obs.ObserveDuration()

		m := msg{
			Protocol:  "choria:adapters:natsstream:output:1",
			Data:      r.Message(),
			Sender:    r.SenderID(),
			Time:      r.Time().UTC(),
			RequestID: r.RequestID(),
		}

		j, err := json.Marshal(m)
		if err != nil {
			sc.log.Warnf("Cannot JSON encode message for publishing to STAN, discarding: %s", err)
			ectr.Inc()
			return
		}

		sc.log.Debugf("Publishing registration data from %s to %s", m.Sender, sc.topic)

		bytes.Add(float64(len(j)))

		// avoids publishing during reconnects while sc.conn could be nil
		sc.mu.Lock()
		defer sc.mu.Unlock()

		err = sc.conn.Publish(sc.topic, j)
		if err != nil {
			sc.log.Warnf("Could not publish message to STAN %s, discarding: %s", sc.topic, err)
			ectr.Inc()
			return
		}

		ctr.Inc()
	}

	for {
		select {
		case r := <-sc.work:
			transformerf(r)

		case <-ctx.Done():
			sc.disconnect()

			return
		}
	}
}
