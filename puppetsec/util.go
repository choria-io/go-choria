package puppetsec

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	puppetwrapper "github.com/choria-io/go-puppet"
)

var puppet = puppetwrapper.New()

func userSSlDir() (string, error) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		path, err := puppet.Setting("ssldir")
		if err != nil {
			return "", err
		}

		return path, nil
	}

	homedir := os.Getenv("HOME")
	if homedir == "" {
		return "", fmt.Errorf("cannot determine home directory, HOME is not set")
	}

	return filepath.FromSlash(filepath.Join(homedir, ".puppetlabs", "etc", "puppet", "ssl")), nil
}
