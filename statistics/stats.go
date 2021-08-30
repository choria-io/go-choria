package statistics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/nats-io/nats-server/v2/server/pse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/protocol"
)

type ChoriaInfo struct {
	Build      BuildInfo `json:"build"`
	System     SysInfo   `json:"system"`
	ConfigFile string    `json:"config_file"`
	Identity   string    `json:"identity"`
}

type BuildInfo struct {
	Version          string `json:"version"`
	SHA              string `json:"sha"`
	BuildDate        string `json:"build_date"`
	License          string `json:"license"`
	TLS              bool   `json:"tls"`
	Secure           bool   `json:"secure"`
	Go               string `json:"go"`
	MaxBrokerClients int    `json:"max_broker_clients"`
}

type SysInfo struct {
	RSS   int64   `json:"rss"`
	PCPU  float64 `json:"cpu_percent"`
	Cores int     `json:"cpu_cores"`
}

type ChoriaFramework interface {
	Configuration() *config.Config
}

var (
	running = false
	mu      = &sync.Mutex{}
	fw      ChoriaFramework

	buildInfo = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "choria_build_info",
		Help: "Build information about the running server",
	}, []string{"version", "sha", "identity"})
)

// Start starts serving exp stats and metrics on the configured statistics port
func Start(cfw ChoriaFramework, handler http.Handler) {
	mu.Lock()
	defer mu.Unlock()

	fw = cfw
	cfg := fw.Configuration()
	port := cfg.Choria.StatsPort

	if port == 0 {
		log.Infof("Statistics gathering disabled, set plugin.choria.stats_port")
		return
	}

	bi := util.BuildInfo()
	prometheus.MustRegister(buildInfo)
	buildInfo.WithLabelValues(bi.Version(), bi.SHA(), fw.Configuration().Identity).Inc()

	if !running {
		log.Infof("Starting statistic reporting Prometheus statistics on http://%s:%d/choria/", cfg.Choria.StatsListenAddress, port)

		if handler == nil {
			http.HandleFunc("/choria/", handleRoot)
			http.Handle("/choria/prometheus", promhttp.Handler())
			http.Handle("/choria/metrics", promhttp.Handler())

			go http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Choria.StatsListenAddress, port), nil)
		} else {
			hh := handler.(*http.ServeMux)
			hh.HandleFunc("/choria/", handleRoot)
			hh.Handle("/choria/prometheus", promhttp.Handler())
			hh.Handle("/choria/metrics", promhttp.Handler())
		}

		running = true
	}
}

func SystemInfo() ChoriaInfo {
	var rss, vss int64
	var pcpu float64

	pse.ProcUsage(&pcpu, &rss, &vss)

	bi := util.BuildInfo()

	return ChoriaInfo{
		ConfigFile: fw.Configuration().ConfigFile,
		Identity:   fw.Configuration().Identity,
		Build: BuildInfo{
			Version:          bi.Version(),
			SHA:              bi.SHA(),
			BuildDate:        bi.BuildDate(),
			License:          bi.License(),
			TLS:              bi.HasTLS(),
			Secure:           protocol.IsSecure(),
			Go:               runtime.Version(),
			MaxBrokerClients: bi.MaxBrokerClients(),
		},
		System: SysInfo{
			RSS:   rss,
			PCPU:  pcpu,
			Cores: runtime.NumCPU(),
		},
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	j, err := json.Marshal(SystemInfo())
	if err != nil {
		j = []byte(fmt.Sprintf(`{"error":%s}`, err))
	}

	fmt.Fprint(w, string(j))
}
