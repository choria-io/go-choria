package mcollective

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// UserConfig determines what is the active config file for a user
// TODO: windows
func UserConfig() string {
	usr, _ := user.Current()

	homeCfg := filepath.Join(usr.HomeDir, ".mcollective")

	if FileExist(homeCfg) {
		return homeCfg
	}

	return filepath.Join("/etc/puppetlabs/mcollective/client.cfg")
}

// FileExist checks if a file exist on disk
func FileExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

// StrToBool converts a typical mcollective boolianish string to bool
func StrToBool(s string) (bool, error) {
	clean := strings.TrimSpace(s)

	if regexp.MustCompile(`(?i)^(1|yes|true|y|t)$`).MatchString(clean) {
		return true, nil
	}

	if regexp.MustCompile(`(?i)^(0|no|false|n|f)$`).MatchString(clean) {
		return false, nil
	}

	return false, errors.New("Cannot convert string value '" + clean + "' into a boolean.")
}

// SliceGroups takes a slice of words and make new chunks of given size
// and call the function with the sub slice.  If there are not enough
// items in the input slice empty strings will pad the last group
func SliceGroups(input []string, size int, fn func(group []string)) {
	// how many to add
	padding := size - (len(input) % size)

	if padding != size {
		p := []string{}

		for i := 0; i <= padding; i++ {
			p = append(p, "")
		}

		input = append(input, p...)
	}

	// how many chunks we're making
	count := len(input) / size

	for i := 0; i < count; i++ {
		chunk := input[i*size : i*size+size]
		fn(chunk)
	}
}

// StringHostsToServers converts an array of servers like host:123 into an array of Server structs
func StringHostsToServers(hosts []string, scheme string) (servers []Server, err error) {
	for _, s := range hosts {
		host, sport, err := net.SplitHostPort(s)
		if err != nil {
			return servers, fmt.Errorf("could not parse host %s: %s", s, err.Error())
		}

		port, err := strconv.Atoi(sport)
		if err != nil {
			return servers, fmt.Errorf("could not host port %s: %s", s, err.Error())
		}

		server := Server{
			Host:   host,
			Port:   port,
			Scheme: scheme,
		}

		servers = append(servers, server)
	}

	return
}
