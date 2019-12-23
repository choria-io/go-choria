package identity

import (
	"regexp"
	"strings"
)

// Match identities on a OR basis, since nodes have only 1 identity
func Match(needles []string, certname string) bool {
	for _, needle := range needles {
		if strings.HasPrefix(needle, "/") && strings.HasSuffix(needle, "/") {
			needle = strings.TrimPrefix(needle, "/")
			needle = strings.TrimSuffix(needle, "/")
			if matched, _ := regexp.MatchString(needle, certname); matched {
				return true
			}

			continue
		}

		if needle == certname {
			return true
		}
	}

	return false
}
