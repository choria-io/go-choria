package statistics

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var running = false
var mu = &sync.Mutex{}

// Start starts serving exp stats and metrics on the configured statistics port
func Start(config *choria.Config, handler http.Handler) {
	mu.Lock()
	defer mu.Unlock()

	port := config.Choria.StatsPort

	if port == 0 {
		log.Infof("Statistics gathering disabled, set plugin.choria.stats_port")
		return
	}

	if !running {
		log.Infof("Starting statistic reporting on port %d /choria/metrics", port)

		if handler == nil {
			http.Handle("/choria/prometheus", promhttp.Handler())

			go http.ListenAndServe(fmt.Sprintf("%s:%d", config.Choria.StatsListenAddress, port), nil)
		} else {
			hh := handler.(*http.ServeMux)
			hh.Handle("/choria/prometheus", promhttp.Handler())
		}

		running = true
	}
}
