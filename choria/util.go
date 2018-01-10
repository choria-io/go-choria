package choria

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// UserConfig determines what is the active config file for a user
// TODO: windows
func UserConfig() string {
	home, _ := HomeDir()

	if home != "" {
		homeCfg := filepath.Join(home, ".mcollective")

		if FileExist(homeCfg) {
			return homeCfg
		}
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
			Host:   strings.TrimSpace(host),
			Port:   port,
			Scheme: scheme,
		}

		servers = append(servers, server)
	}

	return
}

// HomeDir determines the home location without using the user package or requiring cgo
//
// On Unix it needs HOME set and on windows HOMEDRIVE and HOMEDIR
func HomeDir() (string, error) {
	if runtime.GOOS == "windows" {
		drive := os.Getenv("HOMEDRIVE")
		home := os.Getenv("HOMEDIR")

		if home == "" || drive == "" {
			return "", fmt.Errorf("Cannot determine home dir, ensure HOMEDRIVE and HOMEDIR is set")
		}

		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEDIR")), nil
	}

	home := os.Getenv("HOME")

	if home == "" {
		return "", fmt.Errorf("Cannot determine home dir, ensure HOME is set")
	}

	return home, nil

}
