package config

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	confkey "github.com/choria-io/go-config/confkey"
)

// DNSFQDN attempts to find the FQDN using DNS resolution
func DNSFQDN() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

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

// can be used to extract the parsed settings
func parseDotConfFile(plugin string, conf *Config, target interface{}) error {
	cfgPath := filepath.Join(conf.dotdDir(), fmt.Sprintf("%s.cfg", plugin))
	if _, err := os.Stat(cfgPath); err == nil {
		err = parseConfig(cfgPath, target, fmt.Sprintf("plugin.%s", plugin), conf.rawOpts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (conf *Config) parseAllDotCfg() error {
	files, err := ioutil.ReadDir(conf.dotdDir())
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".cfg") {
			base := path.Base(file.Name())
			var target interface{}

			if base == "choria.cfg" {
				target = conf.Choria
			}

			plugin := strings.TrimSuffix(base, filepath.Ext(base))
			err := parseDotConfFile(plugin, conf, target)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// parse a config file and fill in the given config structure based on its tags
func parseConfig(path string, config interface{}, prefix string, found map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	parseConfigContents(file, config, prefix, found)

	return nil
}

func parseConfigContents(content io.Reader, config interface{}, prefix string, found map[string]string) {
	scanner := bufio.NewScanner(content)
	itemr := regexp.MustCompile(`(.+?)\s*=\s*(.+)`)
	skipr := regexp.MustCompile(`^#|^$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !skipr.MatchString(line) {
			if itemr.MatchString(line) {
				matches := itemr.FindStringSubmatch(line)
				var key string

				if prefix == "" {
					key = matches[1]
				} else {
					key = prefix + "." + matches[1]
				}

				if config != nil {
					// errors here are normal since items for Choria and Config are in the same file
					confkey.SetStructFieldWithKey(config, key, matches[2])
				}

				found[key] = matches[2]
			}
		}
	}
}
