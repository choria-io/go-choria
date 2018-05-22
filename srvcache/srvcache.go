package srvcache

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type query struct {
	service string
	proto   string
	name    string
}

type entry struct {
	cname string
	addrs []*net.SRV
	time  time.Time
}

var cache = make(map[query]entry)
var mu = &sync.Mutex{}
var maxage = time.Duration(5 * time.Second)

// LookupSRV is a wrapper around net.LookupSRV that does a 5 second cache
func LookupSRV(service string, proto string, name string, resolver func(string, string, string) (string, []*net.SRV, error)) (string, []*net.SRV, error) {
	mu.Lock()
	defer mu.Unlock()

	var err error

	q := query{service, proto, name}

	cname, addrs := retrieve(q)
	if addrs == nil {
		cname, addrs, err = resolver("", "", name)
	}

	store(q, cname, addrs)

	srvctr.WithLabelValues(identity).Inc()

	return cname, addrs, err
}

func store(q query, cname string, addrs []*net.SRV) {
	cache[q] = entry{
		cname: cname,
		addrs: addrs,
		time:  time.Now(),
	}
}

func retrieve(q query) (string, []*net.SRV) {
	entry, found := cache[q]
	if !found {
		log.Debugf("SRV cache miss on SRV record %#v in cache", q)
		srvmiss.WithLabelValues(identity).Inc()
		return "", nil
	}

	if time.Now().Before(entry.time.Add(maxage)) {
		log.Debugf("SRV cache hit on SRV record %#v", q)
		srvhit.WithLabelValues(identity).Inc()
		return entry.cname, entry.addrs
	}

	log.Debugf("SRV cache miss on SRV record %#v due to age of the record", q)

	delete(cache, q)

	srvmiss.WithLabelValues(identity).Inc()

	return "", nil
}
