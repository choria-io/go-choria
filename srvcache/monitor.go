package srvcache

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

var identity string

var srvctr = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "choria_dns_srv_lookups",
	Help: "Number of SRV queries performed",
}, []string{"identity"})

var srvhit = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "choria_dns_srv_cachehits",
	Help: "Number of SRV lookups served from the cache",
}, []string{"identity"})

var srvmiss = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "choria_dns_srv_cachemiss",
	Help: "Number of SRV cache lookup misses",
}, []string{"identity"})

func init() {
	h, err := os.Hostname()
	if err != nil {
		h = "unknown"
	}

	SetIdentity(h)

	prometheus.MustRegister(srvctr)
	prometheus.MustRegister(srvhit)
	prometheus.MustRegister(srvmiss)
}

// SetIdentity sets the identity to use when reporting cache stats
func SetIdentity(id string) {
	mu.Lock()
	defer mu.Unlock()

	identity = id
}
