package srvcache

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// StringHostsToServers converts an array of servers like host:123 into an array of Servers collection
//
// if an empty scheme is given the string will be parsed by a url parser and the embedded
// scheme will be used, if that does not parse into a valid url then an error will be returned
func StringHostsToServers(hosts []string, scheme string) (servers Servers, err error) {
	instances := make([]Server, len(hosts))

	for i, s := range hosts {
		detectedScheme := scheme
		s = strings.TrimSpace(s)

		u, err := url.Parse(s)
		if err == nil && u.Host != "" {
			s = u.Host

			if scheme == "" {
				detectedScheme = u.Scheme
			}
		}

		host, sport, err := net.SplitHostPort(s)
		if err != nil {
			return servers, fmt.Errorf("could not parse host %s: %s", s, err)
		}

		port, err := strconv.Atoi(sport)
		if err != nil {
			return servers, fmt.Errorf("could not host port %s: %s", s, err)
		}

		server := &BasicServer{
			host:   strings.TrimSpace(host),
			port:   uint16(port),
			scheme: detectedScheme,
		}

		if scheme == "" && detectedScheme == "" {
			return servers, fmt.Errorf("no scheme provided and %s has no scheme", s)
		}

		instances[i] = server
	}

	return NewServers(instances...), nil
}
