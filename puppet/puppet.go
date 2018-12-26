package puppet

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var mu = &sync.Mutex{}
var cache = make(map[string]string)

// FacterStringFact looks up a facter fact, returns "" when unknown
func FacterStringFact(fact string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	value, ok := cache[fact]
	if ok {
		return value, nil
	}

	cmd := FacterCmd()

	if cmd == "" {
		return "", errors.New("could not find your facter command")
	}

	out, err := exec.Command(cmd, fact).Output()
	if err != nil {
		return "", err
	}

	value = strings.Replace(string(out), "\n", "", -1)
	cache[fact] = value

	return value, nil
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
	return AIOCmd("facter", "")
}

// AIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
func AIOCmd(command string, def string) string {
	aioPath := filepath.Join("/opt/puppetlabs/bin", command)

	if runtime.GOOS == "windows" {
		aioPath = filepath.FromSlash(filepath.Join("C:/Program Files/Puppet Labs/Puppet/bin", fmt.Sprintf("%s.bat", command)))
	}

	if _, err := os.Stat(aioPath); err == nil {
		return aioPath
	}

	path, err := exec.LookPath(command)
	if err != nil {
		return def
	}

	return path
}

// Setting retrieves a config setting by shelling out to puppet apply --configprint
func Setting(setting string) (string, error) {
	args := []string{"apply", "--configprint", setting}

	out, err := exec.Command(AIOCmd("puppet", "puppet"), args...).Output()
	if err != nil {
		return "", err
	}

	return strings.Replace(string(out), "\n", "", -1), nil
}
