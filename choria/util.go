package choria

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/internal/util"
)

// UserConfig determines what is the active config file for a user
func UserConfig() string {
	home, _ := util.HomeDir()

	if home != "" {
		// TODO: .choria must go
		for _, n := range []string{".choriarc", ".mcollective"} {
			homeCfg := filepath.Join(home, n)

			if util.FileExist(homeCfg) {
				return homeCfg
			}
		}
	}

	if runtime.GOOS == "windows" {
		return filepath.Join("C:\\", "ProgramData", "choria", "etc", "client.conf")
	}

	if util.FileExist("/etc/choria/client.conf") {
		return "/etc/choria/client.conf"
	}

	if util.FileExist("/usr/local/etc/choria/client.conf") {
		return "/usr/local/etc/choria/client.conf"
	}

	return "/etc/puppetlabs/mcollective/client.cfg"
}

// NewRequestID Creates a new RequestID
func NewRequestID() (string, error) {
	return strings.Replace(util.UniqueID(), "-", "", -1), nil
}

// BuildInfo retrieves build information
func BuildInfo() *build.Info {
	return &build.Info{}
}

// FileExist checks if a file exist
func FileExist(path string) bool {
	return util.FileExist(path)
}
