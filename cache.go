package srvcache

import (
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Resolver performs dns lookup like net.LookupSRV
type Resolver func(service string, proto string, name string) (cname string, addrs []*net.SRV, err error)

// Cache provides a fixed term DNS cache for SRV lookups
type Cache struct {
	identity string
	cache    map[query]*entry
	maxAge   time.Duration
	resolver Resolver
	log      *logrus.Entry

	sync.Mutex
}

type query struct {
	service string
	proto   string
	name    string
}

type entry struct {
	cname   string
	addrs   []*net.SRV
	expTime time.Time
}

// LookupSRV has the same signature as net.LookupSRV but performs caching
func (c *Cache) LookupSRV(service string, proto string, name string) (cname string, addrs []*net.SRV, err error) {
	q := query{service, proto, name}

	cname, addrs = c.retrieve(q)
	if addrs == nil {
		cname, addrs, err = c.resolver("", "", name)
		if err != nil {
			return "", nil, err
		}
	}

	c.store(q, cname, addrs)

	srvctr.WithLabelValues(c.identity).Inc()

	return cname, addrs, err
}

// LookupSrvServers performs a cached SRV lookup and returns a Servers collection
func (c *Cache) LookupSrvServers(service string, proto string, name string, scheme string) (s Servers, err error) {
	_, addrs, err := c.LookupSRV(service, proto, name)
	if err != nil {
		return nil, err
	}

	servers := make([]Server, len(addrs))
	for i, addr := range addrs {
		servers[i] = &BasicServer{host: addr.Target, port: addr.Port, scheme: scheme}
	}

	return NewServers(servers...), nil
}

func (c *Cache) store(q query, cname string, addrs []*net.SRV) {
	c.Lock()
	defer c.Unlock()

	c.cache[q] = &entry{
		cname:   cname,
		addrs:   addrs,
		expTime: time.Now().Add(c.maxAge),
	}
}

func (c *Cache) retrieve(q query) (string, []*net.SRV) {
	c.Lock()
	defer c.Unlock()

	entry, found := c.cache[q]
	if !found {
		c.log.Debugf("SRV cache miss on SRV record %#v in cache", q)
		srvmiss.WithLabelValues(c.identity).Inc()
		return "", nil
	}

	if time.Now().Before(entry.expTime) {
		c.log.Debugf("SRV cache hit on SRV record %#v", q)
		srvhit.WithLabelValues(c.identity).Inc()
		return entry.cname, entry.addrs
	}

	c.log.Debugf("SRV cache miss on SRV record %#v due to age of the record", q)

	delete(c.cache, q)

	srvmiss.WithLabelValues(c.identity).Inc()

	return "", nil
}
