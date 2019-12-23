package classes

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...interface{})
}

// MatchFile classes on a AND basis
func MatchFile(needles []string, source string, log Logger) bool {
	classes, err := ReadClasses(source)
	if err != nil {
		log.Warnf("Could not parse classes file %s: %s", source, err)
		return false
	}

	matched := 0
	failed := 0

	for _, needle := range needles {
		if strings.HasPrefix(needle, "/") && strings.HasSuffix(needle, "/") {
			needle = strings.TrimPrefix(needle, "/")
			needle = strings.TrimSuffix(needle, "/")

			if hasClassMatching(needle, classes) {
				matched++
			} else {
				failed++
			}

			continue
		}

		if hasClass(needle, classes) {
			matched++
		} else {
			failed++
		}
	}

	return failed == 0 && matched > 0
}

func hasClassMatching(needle string, stack []string) bool {
	for _, class := range stack {
		if match, _ := regexp.MatchString(needle, class); match {
			return true
		}
	}

	return false
}

func hasClass(needle string, stack []string) bool {
	for _, class := range stack {
		if class == needle {
			return true
		}
	}

	return false
}

// ReadClasses reads a given file and attempts to parse it as a typical classes file
func ReadClasses(file string) ([]string, error) {
	classes := []string{}

	fh, err := os.Open(file)
	if err != nil {
		return classes, err
	}

	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		classes = append(classes, scanner.Text())
	}

	return classes, nil
}
