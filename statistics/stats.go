package statistics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-protocol/protocol"
	"github.com/nats-io/nats-server/v2/server/pse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type cinfo struct {
	Build      buildinfo `json:"build"`
	System     sysinfo   `json:"system"`
	ConfigFile string    `json:"config_file"`
	Identity   string    `json:"identity"`
}

type buildinfo struct {
	Version          string `json:"version"`
	SHA              string `json:"sha"`
	BuildDate        string `json:"build_date"`
	License          string `json:"license"`
	TLS              bool   `json:"tls"`
	Secure           bool   `json:"secure"`
	Go               string `json:"go"`
	MaxBrokerClients int    `json:"max_broker_clients"`
}

type sysinfo struct {
	RSS   int64   `json:"rss"`
	PCPU  float64 `json:"cpu_percent"`
	Cores int     `json:"cpu_cores"`
}

var (
	running = false
	mu      = &sync.Mutex{}
	cfg     *config.Config

	buildInfo = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_build_info",
		Help: "Build information about the running server",
	}, []string{"version", "sha"})
)

// Start starts serving exp stats and metrics on the configured statistics port
func Start(config *config.Config, handler http.Handler) {
	mu.Lock()
	defer mu.Unlock()

	cfg = config
	port := config.Choria.StatsPort

	if port == 0 {
		log.Infof("Statistics gathering disabled, set plugin.choria.stats_port")
		return
	}

	prometheus.MustRegister(buildInfo)
	buildInfo.WithLabelValues(build.Version, build.SHA).Inc()

	if !running {
		log.Infof("Starting statistic reporting Prometheus statistics on http://%s:%d/choria/", config.Choria.StatsListenAddress, port)

		if handler == nil {
			http.HandleFunc("/choria/", handleRoot)
			http.Handle("/choria/prometheus", promhttp.Handler())

			go http.ListenAndServe(fmt.Sprintf("%s:%d", config.Choria.StatsListenAddress, port), nil)
		} else {
			hh := handler.(*http.ServeMux)
			hh.HandleFunc("/choria/", handleRoot)
			hh.Handle("/choria/prometheus", promhttp.Handler())
		}

		running = true
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	var rss, vss int64
	var pcpu float64

	pse.ProcUsage(&pcpu, &rss, &vss)

	sinfo := cinfo{
		ConfigFile: cfg.ConfigFile,
		Identity:   cfg.Identity,
		Build: buildinfo{
			Version:          build.Version,
			SHA:              build.SHA,
			BuildDate:        build.BuildDate,
			License:          build.License,
			TLS:              build.HasTLS(),
			Secure:           protocol.IsSecure(),
			Go:               runtime.Version(),
			MaxBrokerClients: build.MaxBrokerClients(),
		},
		System: sysinfo{
			RSS:   rss,
			PCPU:  pcpu,
			Cores: runtime.NumCPU(),
		},
	}

	j, err := json.Marshal(sinfo)
	if err != nil {
		j = []byte(fmt.Sprintf(`{"error":%s}`, err))
	}

	fmt.Fprintf(w, string(j))
}
