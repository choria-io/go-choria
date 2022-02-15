// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/AlecAivazis/survey/v2"
	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/build"
	"github.com/gofrs/uuid"
	"github.com/olekukonko/tablewriter"
)

var ConstantBackOffForTests = backoff.Policy{
	Millis: []int{25},
}

// BuildInfo retrieves build information
func BuildInfo() *build.Info {
	return &build.Info{}
}

// GovernorSubject the subject to use for choria managed Governors within a collective
func GovernorSubject(name string, collective string) string {
	return fmt.Sprintf("%s.governor.%s", collective, name)
}

// UserConfig determines what is the active config file for a user
func UserConfig() string {
	home, _ := HomeDir()

	if home != "" {
		for _, n := range []string{".choriarc", ".mcollective"} {
			homeCfg := filepath.Join(home, n)

			if FileExist(homeCfg) {
				return homeCfg
			}
		}
	}

	if runtime.GOOS == "windows" {
		return filepath.Join("C:\\", "ProgramData", "choria", "etc", "client.conf")
	}

	if FileExist("/etc/choria/client.conf") {
		return "/etc/choria/client.conf"
	}

	if FileExist("/usr/local/etc/choria/client.conf") {
		return "/usr/local/etc/choria/client.conf"
	}

	return "/etc/puppetlabs/mcollective/client.cfg"
}

// FileExist checks if a file exist on disk
func FileExist(path string) bool {
	if path == "" {
		return false
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

// HomeDir determines the home location without using the user package or requiring cgo
//
// On Unix it needs HOME set and on windows HOMEDRIVE and HOMEDIR
func HomeDir() (string, error) {
	if runtime.GOOS == "windows" {
		drive := os.Getenv("HOMEDRIVE")
		home := os.Getenv("HOMEDIR")

		if home == "" || drive == "" {
			return "", fmt.Errorf("cannot determine home dir, ensure HOMEDRIVE and HOMEDIR is set")
		}

		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEDIR")), nil
	}

	home := os.Getenv("HOME")

	if home == "" {
		return "", fmt.Errorf("cannot determine home dir, ensure HOME is set")
	}

	return home, nil

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

// StringInList checks if match is in list
func StringInList(list []string, match string) bool {
	for _, i := range list {
		if i == match {
			return true
		}
	}

	return false
}

// InterruptibleSleep sleep for the duration in a way that can be interrupted by the context.
// An error is returned if the context cancels the sleep
func InterruptibleSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("sleep interrupted by context")
	}
}

// UniqueID creates a new unique ID, usually a v4 uuid, if that fails a random string based ID is made
func UniqueID() (id string) {
	uuid, err := uuid.NewV4()
	if err == nil {
		return uuid.String()
	}

	parts := []string{}
	parts = append(parts, randStringRunes(8))
	parts = append(parts, randStringRunes(4))
	parts = append(parts, randStringRunes(4))
	parts = append(parts, randStringRunes(12))

	return strings.Join(parts, "-")
}

func randStringRunes(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}

// LongestString determines the length of the longest string in list, capped at max
func LongestString(list []string, max int) int {
	longest := 0
	for _, i := range list {
		if len(i) > longest {
			longest = len(i)
		}

		if max != 0 && longest > max {
			return max
		}
	}

	return longest
}

// ParagraphPadding pads paragraph with padding spaces
func ParagraphPadding(paragraph string, padding int) string {
	parts := strings.Split(paragraph, "\n")
	ps := fmt.Sprintf("%"+strconv.Itoa(padding)+"s", " ")

	for i := range parts {
		parts[i] = ps + parts[i]
	}

	return strings.Join(parts, "\n")
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

// SliceVerticalGroups takes a slice of words and make new chunks of given size
// and call the function with the sub slice.  The results are ordered for
// vertical alignment.  If there are not enough items in the input slice empty
// strings will pad the last group
func SliceVerticalGroups(input []string, size int, fn func(group []string)) {
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
		chunk := []string{}
		for s := 0; s < size; s++ {
			chunk = append(chunk, input[i+s*count])
		}
		fn(chunk)
	}
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

func FileIsRegular(path string) bool {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	if !stat.Mode().IsRegular() {
		return false
	}

	return true
}

func FileIsDir(path string) bool {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	if !stat.IsDir() {
		return false
	}

	return true
}

func UniqueStrings(items []string, shouldSort bool) []string {
	keys := make(map[string]struct{})
	result := []string{}
	for _, i := range items {
		_, ok := keys[i]
		if !ok {
			keys[i] = struct{}{}
			result = append(result, i)
		}
	}

	if shouldSort {
		sort.Strings(result)
	}

	return result
}

// ExpandPath expands a path that starts in ~ to the users homedir
func ExpandPath(p string) (string, error) {
	a := strings.TrimSpace(p)
	if a[0] == '~' {
		home, err := HomeDir()
		if err != nil {
			return "", err
		}
		a = strings.Replace(a, "~", home, 1)
	}
	return a, nil
}

// NewMarkdownTable makes a new table writer formatted to be valid markdown
func NewMarkdownTable(hdr ...string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader(hdr)

	return table
}

// StringsMapKeys returns the keys from a map[string]string in sorted order
func StringsMapKeys(data map[string]string) []string {
	keys := make([]string, len(data))
	i := 0
	for k := range data {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	return keys
}

// IterateStringsMap iterates a map[string]string in key sorted order
func IterateStringsMap(data map[string]string, cb func(k string, v string)) {
	for _, k := range StringsMapKeys(data) {
		cb(k, data[k])
	}
}

// DumpMapStrings shows k: v of a map[string]string left padded by int, the k will be right aligned and value left aligned
func DumpMapStrings(data map[string]string, leftPad int) {
	longest := LongestString(StringsMapKeys(data), 0) + leftPad

	IterateStringsMap(data, func(k, v string) {
		fmt.Printf("%s: %s\n", strings.Repeat(" ", longest-len(k))+k, v)
	})
}

// DumpJSONIndent dumps data to stdout as indented JSON
func DumpJSONIndent(data interface{}) error {
	j, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))

	return nil
}

// RenderDuration create a string similar to what %v on a duration would but it supports years, months, etc.
// Being that days and years are influenced by leap years and such it will never be 100% accurate but for
// feedback on the terminal its sufficient
func RenderDuration(d time.Duration) string {
	if d == math.MaxInt64 {
		return "never"
	}

	if d == 0 {
		return "forever"
	}

	tsecs := d / time.Second
	tmins := tsecs / 60
	thrs := tmins / 60
	tdays := thrs / 24
	tyrs := tdays / 365

	if tyrs > 0 {
		return fmt.Sprintf("%dy%dd%dh%dm%ds", tyrs, tdays%365, thrs%24, tmins%60, tsecs%60)
	}

	if tdays > 0 {
		return fmt.Sprintf("%dd%dh%dm%ds", tdays, thrs%24, tmins%60, tsecs%60)
	}

	if thrs > 0 {
		return fmt.Sprintf("%dh%dm%ds", thrs, tmins%60, tsecs%60)
	}

	if tmins > 0 {
		return fmt.Sprintf("%dm%ds", tmins, tsecs%60)
	}

	return fmt.Sprintf("%.2fs", d.Seconds())
}

// ParseDuration is an extended version of go duration parsing that
// also supports w,W,d,D,M,Y,y in addition to what go supports
func ParseDuration(dstr string) (dur time.Duration, err error) {
	dstr = strings.TrimSpace(dstr)

	if len(dstr) <= 0 {
		return dur, nil
	}

	ls := len(dstr)
	di := ls - 1
	unit := dstr[di:]

	switch unit {
	case "w", "W":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*7*24) * time.Hour

	case "d", "D":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*24) * time.Hour
	case "M":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*24*30) * time.Hour
	case "Y", "y":
		val, err := strconv.ParseFloat(dstr[:di], 32)
		if err != nil {
			return dur, err
		}

		dur = time.Duration(val*24*365) * time.Hour
	case "s", "S", "m", "h", "H":
		dur, err = time.ParseDuration(dstr)
		if err != nil {
			return dur, err
		}

	default:
		return dur, fmt.Errorf("invalid time unit %s", unit)
	}

	return dur, nil
}

// PromptForConfirmation asks for confirmation on the CLI
func PromptForConfirmation(format string, a ...interface{}) (bool, error) {
	ans := false
	err := survey.AskOne(&survey.Confirm{
		Message: fmt.Sprintf(format, a...),
		Default: ans,
	}, &ans)

	return ans, err
}

// IsPrintable determines if a string is printable
func IsPrintable(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII || !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// Base64IfNotPrintable returns a string value that's either the given value or base64 encoded if not IsPrintable()
func Base64IfNotPrintable(val []byte) string {
	if IsPrintable(string(val)) {
		return string(val)
	}

	return base64.StdEncoding.EncodeToString(val)
}

func tStringsJoin(s []string) string {
	return strings.Join(s, ", ")
}

func tBase64Encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func tBase64Decode(v string) (string, error) {
	r, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", err
	}

	return string(r), nil
}

func FuncMap(f map[string]interface{}) template.FuncMap {
	fm := map[string]interface{}{
		"Title":        strings.Title,
		"Capitalize":   strings.Title,
		"ToLower":      strings.ToLower,
		"ToUpper":      strings.ToUpper,
		"StringsJoin":  tStringsJoin,
		"Base64Encode": tBase64Encode,
		"Base64Decode": tBase64Decode,
	}

	for k, v := range f {
		fm[k] = v
	}

	return fm
}

func FileHasSha256Sum(path string, sum string) (bool, string, error) {
	s, err := Sha256HashFile(path)
	if err != nil {
		return false, "", err
	}

	return s == sum, s, nil
}

func Sha256HashBytes(c []byte) (string, error) {
	hasher := sha256.New()
	r := bytes.NewReader(c)
	_, err := io.Copy(hasher, r)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func IsExecutableInPath(c string) bool {
	p, err := exec.LookPath(c)
	if err != nil {
		return false
	}

	return p != ""
}

func Sha256HashFile(path string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
