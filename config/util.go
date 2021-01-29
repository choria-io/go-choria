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

	"github.com/choria-io/go-choria/confkey"
	"github.com/choria-io/go-choria/internal/util"
)

// ProjectConfigurationFiles returns any configuration file in the specified directory and their parents directories.
func ProjectConfigurationFiles(path string) ([]string, error) {
	var (
		res []string
		err error
	)

	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
	}

	var parent = filepath.Dir(path)
	if parent != path {
		res, err = ProjectConfigurationFiles(parent)
		if err != nil {
			return nil, err
		}
	}

	config := filepath.Join(path, "choria.conf")
	if util.FileExist(config) {
		res = append(res, config)
	}

	return res, nil
}

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
	if util.FileExist(cfgPath) {
		err := parseConfig(cfgPath, target, fmt.Sprintf("plugin.%s", plugin), conf.rawOpts)
		if err != nil {
			return err
		}

		conf.ParsedFiles = append(conf.ParsedFiles, cfgPath)
	}

	return nil
}

// parseAllDotCfg parses a file like /etc/..../plugin.d/package.cfg as if its full of
// plugin.package.x = y lines and fill in a structure with the results if that structure
// declares its options using the same tag structure as Config.
//
// If the supplied target structure is nil then the only side effect will be that the
// supplied conf will be updated with the raw options so that HasOption() and Option()
func (c *Config) parseAllDotCfg() error {
	dir := c.dotdDir()
	if dir == "" {
		return nil
	}

	if !util.FileIsDir(dir) {
		return nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if ext == ".cfg" || ext == ".conf" {
			base := path.Base(file.Name())
			var target interface{}

			if base == "choria.cfg" {
				target = c.Choria
			}

			plugin := strings.TrimSuffix(base, filepath.Ext(base))
			err = parseDotConfFile(plugin, c, target)
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

	c, ok := config.(*Config)
	if ok {
		c.ParsedFiles = append(c.ParsedFiles, path)
	}

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
