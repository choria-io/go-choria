package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/broker/adapter/ingest"
	"github.com/choria-io/go-choria/broker/adapter/stats"
	"github.com/choria-io/go-choria/broker/adapter/transformer"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type stream struct {
	servers     func() (srvcache.Servers, error)
	clientID    string
	topic       string
	conn        choria.Connector
	log         *log.Entry
	name        string
	adapterName string

	work chan ingest.Adaptable
}

func newStream(name string, work chan ingest.Adaptable, logger *log.Entry) ([]*stream, error) {
	prefix := fmt.Sprintf("plugin.choria.adapter.%s.stream.", name)

	instances, err := strconv.Atoi(cfg.Option(prefix+"workers", "10"))
	if err != nil {
		return nil, fmt.Errorf("%s should be a integer number", prefix+"workers")
	}

	servers := cfg.Option(prefix+"servers", "")

	topic := cfg.Option(prefix+"topic", "")
	if topic == "" {
		topic = name
	}

	var workers []*stream

	for i := 0; i < instances; i++ {
		logger.Infof("Creating NATS JetStream Adapter %s instance %d / %d publishing to message set %s", name, i, instances, topic)

		iname := fmt.Sprintf("%s_%d-%s", name, i, strings.Replace(util.UniqueID(), "-", "", -1))

		st := &stream{
			clientID:    iname,
			topic:       topic,
			name:        fmt.Sprintf("%s.%d", name, i),
			adapterName: name,
			work:        work,
			log:         logger.WithFields(log.Fields{"side": "stream", "instance": i}),
		}

		if servers != "" {
			st.servers = st.resolver(strings.Split(servers, ","))
		} else {
			st.log.Warnf("%s not set, using standard client middleware resolution", prefix+"servers")
			st.servers = fw.MiddlewareServers
		}

		workers = append(workers, st)
	}

	return workers, nil
}

func (sc *stream) resolver(parts []string) func() (srvcache.Servers, error) {
	servers, err := srvcache.StringHostsToServers(parts, "nats")
	return func() (srvcache.Servers, error) {
		return servers, err
	}
}

func (sc *stream) connect(ctx context.Context, cm choria.ConnectionManager) error {
	if ctx.Err() != nil {
		return fmt.Errorf("shutdown called")
	}

	nc, err := fw.NewConnector(ctx, sc.servers, sc.clientID, sc.log)
	if err != nil {
		return fmt.Errorf("could not start JetStream connection: %s", err)
	}

	sc.conn = nc

	sc.log.Infof("%s connected to JetStream", sc.clientID)

	return nil
}

func (sc *stream) disconnect() {
	if sc.conn != nil {
		sc.log.Info("Disconnecting from JetStream")
		sc.conn.Close()
	}
}

func (sc *stream) publisher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	bytes := stats.BytesCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	ectr := stats.ErrorCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	ctr := stats.ReceivedMsgsCtr.WithLabelValues(sc.name, "output", cfg.Identity)
	timer := stats.ProcessTime.WithLabelValues(sc.name, "output", cfg.Identity)
	workqlen := stats.WorkQueueLengthGauge.WithLabelValues(sc.adapterName, cfg.Identity)

	transformerf := func(r ingest.Adaptable) {
		obs := prometheus.NewTimer(timer)
		defer obs.ObserveDuration()
		defer func() { workqlen.Set(float64(len(sc.work))) }()

		j, err := json.Marshal(transformer.TransformToOutput(r, "jetstream"))
		if err != nil {
			sc.log.Warnf("Cannot JSON encode message for publishing to JetStream, discarding: %s", err)
			ectr.Inc()
			return
		}

		sc.log.Debugf("Publishing registration data from %s to %s", r.SenderID(), sc.topic)

		bytes.Add(float64(len(j)))

		err = sc.conn.PublishRaw(strings.ReplaceAll(sc.topic, "%s", r.SenderID()), j)
		if err != nil {
			sc.log.Warnf("Could not publish message to JetStream %s, discarding: %s", sc.topic, err)
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
