package statistics

import (
	"net/http"
	"sync"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/sirupsen/logrus"
)

var registry = metrics.NewRegistry()
var mu = &sync.Mutex{}
var data = make(map[string]interface{})
var log = logrus.WithFields(logrus.Fields{})

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
