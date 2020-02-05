package srvcache

import (
	"github.com/prometheus/client_golang/prometheus"
)

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
	prometheus.Register(srvctr)
	prometheus.Register(srvhit)
	prometheus.Register(srvmiss)
}
