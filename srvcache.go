// Package srvcache provides a caching SRV lookup service that creates a short term
// cache of SRV answers - it does not comply with DNS protocols like the timings
// set by DNS servers, its more a short term - think 5 seconds - buffer to avoid
// hitting the dns servers repeatedly
package srvcache

import (
	"time"

	"github.com/sirupsen/logrus"
)

// New creates a new Cache
func New(identity string, maxAge time.Duration, resolver Resolver, log *logrus.Entry) *Cache {
	return &Cache{
		identity: identity,
		cache:    make(map[query]*entry),
		maxAge:   maxAge,
		resolver: resolver,
		log:      log,
	}
}
