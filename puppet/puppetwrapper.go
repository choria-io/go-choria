package puppet

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
)

// Wrapper provides ways to interact with Puppet and Facter
type Wrapper struct {
	cache map[string]string
	sync.Mutex
}

// New creates a new wrapper
func New() *Wrapper {
	return &Wrapper{
		cache: make(map[string]string),
	}
}

func (p *Wrapper) read(f string) (string, bool) {
	p.Lock()
	defer p.Unlock()

	f, ok := p.cache[f]

	return f, ok
}

func (p *Wrapper) store(f string, val string) {
	p.Lock()
	defer p.Unlock()

	p.cache[f] = val
}

// FacterStringFact looks up a facter fact, returns "" when unknown
func (p *Wrapper) FacterStringFact(fact string) (string, error) {
	value, ok := p.read(fact)
	if ok {
		return value, nil
	}

	cmd := p.FacterCmd()

	if cmd == "" {
		return "", errors.New("could not find your facter command")
	}

	out, err := exec.Command(cmd, fact).Output()
	if err != nil {
		return "", err
	}

	value = strings.Replace(string(out), "\n", "", -1)
	value = strings.Replace(value, "\r", "", -1)

	p.store(fact, value)

	return value, nil
}

// FacterFQDN determines the machines fqdn by querying facter.  Returns "" when unknown
func (p *Wrapper) FacterFQDN() (string, error) {
	return p.FacterStringFact("networking.fqdn")
}

// FacterDomain determines the machines domain by querying facter. Returns "" when unknown
func (p *Wrapper) FacterDomain() (string, error) {
	return p.FacterStringFact("networking.domain")
}

// FacterCmd finds the path to facter using first AIO path then a `which` like command
func (p *Wrapper) FacterCmd() string {
	return p.AIOCmd("facter", "")
}

// AIOCmd looks up a command in the AIO paths, if it's not there
// it will try PATH and finally return a default if not in PATH
func (p *Wrapper) AIOCmd(command string, def string) string {
	aioPath := filepath.Join("/opt/puppetlabs/bin", command)

	if runtime.GOOS == "windows" {
		aioPath = filepath.FromSlash(filepath.Join("C:/Program Files/Puppet Labs/Puppet/bin", fmt.Sprintf("%s.bat", command)))
	}

	if util.FileExist(aioPath) {
		return aioPath
	}

	path, err := exec.LookPath(command)
	if err != nil {
		return def
	}

	return path
}

// Setting retrieves a config setting by shelling out to puppet apply --configprint
func (p *Wrapper) Setting(setting string) (string, error) {
	args := []string{"apply", "--environment", "production", "--configprint", setting}

	out, err := exec.Command(p.AIOCmd("puppet", "puppet"), args...).Output()
	if err != nil {
		return "", fmt.Errorf("could not run 'puppet %s': %s: %s", strings.Join(args, " "), err, out)
	}

	value := strings.Replace(string(out), "\n", "", -1)
	value = strings.Replace(value, "\r", "", -1)

	return value, nil
}
