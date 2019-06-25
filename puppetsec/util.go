package puppetsec

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	puppetwrapper "github.com/choria-io/go-puppet"
)

var puppet = puppetwrapper.New()

func userSSlDir() (string, error) {
	if os.Geteuid() == 0 {
		path, err := puppet.Setting("ssldir")
		if err != nil {
			return "", err
		}

		return path, nil
	}

	homedir := os.Getenv("HOME")

	if runtime.GOOS == "windows" {
		if os.Getenv("HOMEDRIVE") == "" || os.Getenv("HOMEPATH") == "" {
			return "", errors.New("cannot determine home dir while looking for SSL Directory, no HOMEDRIVE or HOMEPATH environment is set.  Please set HOME or configure plugin.choria.ssldir")
		}

		homedir = filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
	}

	return filepath.FromSlash(filepath.Join(homedir, ".puppetlabs", "etc", "puppet", "ssl")), nil
}
