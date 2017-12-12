package statistics

import (
	"expvar"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/sirupsen/logrus"
)

var registry = metrics.NewRegistry()
var mu = &sync.Mutex{}
var data = make(map[string]interface{})
var log = logrus.WithFields(logrus.Fields{})
var running = false

func Counter(name string) metrics.Counter {
	mu.Lock()
	defer mu.Unlock()

	return getOrCreate(name, func() interface{} {
		return metrics.NewCounter()
	}).(metrics.Counter)
}

func Timer(name string) metrics.Timer {
	mu.Lock()
	defer mu.Unlock()

	return getOrCreate(name, func() interface{} {
		return metrics.NewTimer()
	}).(metrics.Timer)
}

func getOrCreate(name string, create func() interface{}) interface{} {
	c, ok := data[name]
	if ok {
		return c
	}

	m := create()

	data[name] = m
	registry.Register(name, m)

	return m
}

func HTTPHandler() http.Handler {
	return exp.ExpHandler(registry)
}

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

		expvar.NewString("version").Set(build.Version)
		expvar.NewString("build_sha").Set(build.SHA)
		expvar.NewString("build_date").Set(build.BuildDate)
		expvar.NewString("config").Set(config.ConfigFile)

		pReg := prometheus.NewRegistry()
		pClient := NewPrometheusProvider(registry, "choria", pReg, 1*time.Second)
		go pClient.UpdatePrometheusMetrics()

		if handler == nil {
			http.Handle("/choria/metrics", HTTPHandler())
			http.Handle("/choria/prometheus", promhttp.HandlerFor(pReg, promhttp.HandlerOpts{}))

			go http.ListenAndServe(fmt.Sprintf("%s:%d", config.Choria.StatsListenAddress, port), nil)
		} else {
			hh := handler.(*http.ServeMux)
			hh.Handle("/choria/prometheus", promhttp.HandlerFor(pReg, promhttp.HandlerOpts{}))
			hh.Handle("/choria/metrics", HTTPHandler())
		}

		running = true
	}
}
