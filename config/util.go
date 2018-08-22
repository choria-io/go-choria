package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// DNSFQDN attempts to find the FQDN using DNS resolution
func DNSFQDN() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	fmt.Println(hostname)

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return "", err
			}

			hosts, err := net.LookupAddr(string(ip))
			if err != nil || len(hosts) == 0 {
				return "", err
			}

			fqdn := hosts[0]

			// return fqdn without trailing dot
			return strings.TrimSuffix(fqdn, "."), nil
		}
	}

	return "", fmt.Errorf("could not resolve FQDN using DNS")
}
