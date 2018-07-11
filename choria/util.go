package choria

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/choria-io/go-choria/puppet"
	"github.com/choria-io/go-choria/srvcache"
)

// UserConfig determines what is the active config file for a user
// TODO: windows
func UserConfig() string {
	home, _ := HomeDir()

	if home != "" {
		for _, n := range []string{".choria", ".mcollective"} {
			homeCfg := filepath.Join(home, n)

			if FileExist(homeCfg) {
				return homeCfg
			}
		}
	}

	if FileExist("/etc/choria/client.cfg") {
		return "/etc/choria/client.cfg"
	}

	return "/etc/puppetlabs/mcollective/client.cfg"
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

	return false, fmt.Errorf("cannot convert string value '%s' into a boolean", clean)
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
//
// if an empty scheme is given the string will be parsed by a url parser and the embedded
// scheme will be used, if that does not parse into a valid url then an error will be returned
func StringHostsToServers(hosts []string, scheme string) (servers []srvcache.Server, err error) {
	for _, s := range hosts {
		detectedScheme := scheme

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

		server := srvcache.Server{
			Host:   strings.TrimSpace(host),
			Port:   port,
			Scheme: detectedScheme,
		}

		if scheme == "" && detectedScheme == "" {
			return servers, fmt.Errorf("no scheme provided and %s has no scheme", s)
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

// FacterStringFact looks up a facter fact, returns "" when unknown
func FacterStringFact(fact string) (string, error) {
	return puppet.FacterStringFact(fact)
}

// FacterFQDN determines the machines fqdn by querying facter.  Returns "" when unknown
func FacterFQDN() (string, error) {
	return FacterStringFact("networking.fqdn")
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func FacterDomain() (string, error) {
	return FacterStringFact("networking.domain")
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
func FacterCmd() string {
	return puppet.AIOCmd("facter", "")
}

// PuppetAIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
//
// TODO: windows support
func PuppetAIOCmd(command string, def string) string {
	return puppet.AIOCmd(command, def)
}

// PuppetSetting retrieves a config setting by shelling out to puppet apply --configprint
func PuppetSetting(setting string) (string, error) {
	args := []string{"apply", "--configprint", setting}

	out, err := exec.Command(PuppetAIOCmd("puppet", "puppet"), args...).Output()
	if err != nil {
		return "", err
	}

	return strings.Replace(string(out), "\n", "", -1), nil
}

// MatchAnyRegex checks str against a list of possible regex, if any match true is returned
func MatchAnyRegex(str []byte, regex []string) bool {
	for _, reg := range regex {
		if matched, _ := regexp.Match(reg, str); matched {
			return true
		}
	}

	return false
}
