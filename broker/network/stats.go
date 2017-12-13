package network

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/nats-io/gnatsd/server"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	ConnectionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_connections",
		Help: "Current connections on the network broker",
	})

	TotalConnectionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_total_connections",
		Help: "Total connections received since start",
	})

	RoutesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_routes",
		Help: "Current active routes to other brokers",
	})

	RemotesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_remotes",
		Help: "Current active connections to other brokers",
	})

	InMsgsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_in_msgs",
		Help: "Messages received by the network broker",
	})

	OutMsgsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_out_msgs",
		Help: "Messages sent by the network broker",
	})

	InBytesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_in_bytes",
		Help: "Total size of messages received by the network broker",
	})

	OutBytesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_out_bytes",
		Help: "Total size of messages sent by the network broker",
	})

	SlowConsumerGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_slow_consumers",
		Help: "Total number of clients who were considered slow consumers",
	})

	SubscriptionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "choria_network_subscriptions",
		Help: "Number of active subscriptions to subjects on this broker",
	})
)

func init() {
	prometheus.MustRegister(ConnectionsGauge)
	prometheus.MustRegister(TotalConnectionsGauge)
	prometheus.MustRegister(RoutesGauge)
	prometheus.MustRegister(RemotesGauge)
	prometheus.MustRegister(InMsgsGauge)
	prometheus.MustRegister(OutMsgsGauge)
	prometheus.MustRegister(InBytesGauge)
	prometheus.MustRegister(OutBytesGauge)
	prometheus.MustRegister(SlowConsumerGauge)
	prometheus.MustRegister(SubscriptionsGauge)
}

func (s *Server) getVarz() (*server.Varz, error) {
	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	url := fmt.Sprintf("http://%s:%d/varz", s.opts.HTTPHost, s.opts.HTTPPort)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get /varz stats: %s", err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Could not get /varz stats: %s", err.Error())
	}

	response := &server.Varz{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, fmt.Errorf("Could not get /varz stats: %s", err.Error())
	}

	return response, nil
}

func (s *Server) publishStats(ctx context.Context, interval time.Duration) {
	if s.opts.HTTPPort == 0 {
		return
	}

	timer := time.NewTimer(interval)

	s.updatePrometheus()

	for {
		select {
		case <-timer.C:
			s.updatePrometheus()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) updatePrometheus() {
	varz, err := s.getVarz()
	if err != nil {
		log.Errorf("Could not publish network broker stats: %s", err.Error())
	}

	ConnectionsGauge.Set(float64(varz.Connections))
	TotalConnectionsGauge.Set(float64(varz.TotalConnections))
	RoutesGauge.Set(float64(varz.Routes))
	RemotesGauge.Set(float64(varz.Remotes))
	InMsgsGauge.Set(float64(varz.InMsgs))
	OutMsgsGauge.Set(float64(varz.OutMsgs))
	InBytesGauge.Set(float64(varz.InBytes))
	OutBytesGauge.Set(float64(varz.OutBytes))
	SlowConsumerGauge.Set(float64(varz.SlowConsumers))
	SubscriptionsGauge.Set(float64(varz.Subscriptions))

}
