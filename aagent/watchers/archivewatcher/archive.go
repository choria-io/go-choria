// Copyright (c) 2021-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	"github.com/choria-io/go-choria/aagent/util"
	"github.com/choria-io/go-choria/aagent/watchers/event"
	"github.com/choria-io/go-choria/aagent/watchers/watcher"
	"github.com/choria-io/go-choria/build"
	iu "github.com/choria-io/go-choria/internal/util"
)

type State int

const (
	Unknown State = iota
	Skipped
	Error
	VerifiedOK
	Downloaded
	VerifyFailed
	MissingCreates
	MissingChecksums

	wtype   = "archive"
	version = "v1"
)

var stateNames = map[State]string{
	Unknown:          "unknown",
	Skipped:          "skipped",
	Error:            "error",
	VerifiedOK:       "verified",
	Downloaded:       "downloaded",
	VerifyFailed:     "verify_failed",
	MissingCreates:   "no_creates",
	MissingChecksums: "no_checksums",
}

type Properties struct {
	// ArchiveChecksum is a sha256 hex string of the archive being downloaded, requires ArchiveChecksumChecksum
	ArchiveChecksum string `mapstructure:"checksum"`
	// Creates is a subdirectory that the tarball will create on untar, it has to create a sub directory
	Creates string
	// Governor is the optional name of a governor to use for concurrency control
	Governor string
	// GovernorTimeout is how long we'll try to access the governor
	GovernorTimeout time.Duration `mapstructure:"governor_timeout"`
	// Insecure skips TLS verification on https downloads (not implemented)
	Insecure bool
	// Password for accessing the source, required when a username is set
	Password string
	// Source is a URL to the file being downloaded, only tar.gz format is supported
	Source string
	// TargetDirectory is the directory where the tarball will be extracted
	TargetDirectory string `mapstructure:"target"`
	// Timeout is how long HTTP operations are allowed to take
	Timeout time.Duration
	// Username is the username to use when downloading, Password is required in addition
	Username string
	// ContentChecksums a file in the archive made using sha256 used for verification of files in the archive after extraction and on every interval check
	ContentChecksums string `mapstructure:"verify"`
	// ContentChecksumsChecksum is a sha256 hex string of the file specified in ContentChecksums
	ContentChecksumsChecksum string `mapstructure:"verify_checksum"`
}

type Watcher struct {
	*watcher.Watcher

	name            string
	machine         model.Machine
	previous        State
	interval        time.Duration
	previousRunTime time.Duration
	previousSource  string
	properties      *Properties

	lastWatch time.Time

	wmu *sync.Mutex
	mu  *sync.Mutex
}

func New(machine model.Machine, name string, states []string, required []model.ForeignMachineState, failEvent string, successEvent string, interval string, ai time.Duration, rawprop map[string]any) (any, error) {
	var err error

	archive := &Watcher{
		name:       name,
		machine:    machine,
		properties: &Properties{},
		lastWatch:  time.Time{},
		wmu:        &sync.Mutex{},
		mu:         &sync.Mutex{},
	}

	archive.Watcher, err = watcher.NewWatcher(name, wtype, ai, states, required, machine, failEvent, successEvent)
	if err != nil {
		return nil, err
	}

	err = archive.setProperties(rawprop)
	if err != nil {
		return nil, fmt.Errorf("could not set properties: %v", err)
	}

	if interval != "" {
		archive.interval, err = iu.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}

		if archive.interval < 10*time.Second {
			return nil, fmt.Errorf("interval %v is too small", archive.interval)
		}
	}

	return archive, nil
}

func (w *Watcher) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.Infof("archive watcher for %s starting", w.name)

	if w.interval != 0 {
		wg.Add(1)
		go w.intervalWatcher(ctx, wg)
	}

	w.performWatch(ctx, false)

	for {
		select {
		case <-w.Watcher.StateChangeC():
			w.performWatch(ctx, true)

		case <-ctx.Done():
			w.Infof("Stopping on context interrupt")
			w.CancelGovernor()
			return
		}
	}
}

func (w *Watcher) verifyCreates() (string, State, error) {
	creates := filepath.Join(w.properties.TargetDirectory, w.properties.Creates)

	if !iu.FileIsDir(creates) {
		return creates, MissingCreates, nil
	}

	if w.properties.ContentChecksums == "" {
		return creates, VerifiedOK, nil
	}

	checksums := filepath.Join(creates, w.properties.ContentChecksums)
	if !iu.FileExist(checksums) {
		w.Errorf("Checksums file %s does not exist in %s, triggering download: %s", checksums, w.properties.ContentChecksums)
		return creates, MissingChecksums, nil
	}

	// TODO: if verify fail on checksum fail of the sha256sums file should I remove the resulting files,
	//  they are probably compromised so should stop being used maybe a flag to control that
	if w.properties.ContentChecksumsChecksum == "" {
		return creates, VerifiedOK, nil
	}

	err := w.verify(creates)
	if err == nil {
		w.Infof("Checksums of %s verified successfully using %s", creates, w.properties.ContentChecksums)
		return creates, VerifiedOK, nil
	}

	w.Errorf("Checksum verification failed, triggering download: %v", err)

	return creates, VerifyFailed, nil
}

func (w *Watcher) watch(ctx context.Context) (state State, err error) {
	if !w.ShouldWatch() {
		return Skipped, nil
	}

	start := time.Now()
	defer func() {
		w.mu.Lock()
		w.previousRunTime = time.Since(start)
		w.mu.Unlock()
	}()

	creates, state, err := w.verifyCreates()
	if err == nil && state == VerifiedOK {
		return state, err
	}

	if w.properties.Governor != "" {
		fin, err := w.EnterGovernor(ctx, w.properties.Governor, w.properties.GovernorTimeout)
		if err != nil {
			w.Errorf("Cannot enter Governor %s: %s", w.properties.Governor, err)
			return Error, err
		}
		defer fin()
	}

	timeout, cancel := context.WithTimeout(ctx, w.properties.Timeout)
	defer cancel()

	tf, err := w.downloadSourceToTemp(timeout)
	if tf != "" {
		defer os.RemoveAll(filepath.Dir(tf))
	}
	if err != nil {
		return Error, fmt.Errorf("download failed: %s", err)
	}
	if tf == "" {
		return Error, fmt.Errorf("unknown error downloading to temporary file")
	}

	td, err := w.extractAndVerifyToTemp(tf)
	if err != nil {
		return Error, fmt.Errorf("archive extraction failed: %s", err)
	}

	if iu.FileExist(creates) {
		err = os.RemoveAll(creates)
		if err != nil {
			return Error, fmt.Errorf("removing current destination failed: %s", err)
		}
	}

	if !iu.FileIsDir(w.properties.TargetDirectory) {
		err = os.MkdirAll(w.properties.TargetDirectory, 0700)
		if err != nil {
			return Error, fmt.Errorf("could not create target directory: %s", err)
		}
	}

	err = os.Rename(filepath.Join(td, w.properties.Creates), creates)
	if err != nil {
		return Error, fmt.Errorf("rename failed: %s", err)
	}

	w.Warnf("Archive %v was deployed successfully to %v", w.properties.Source, w.properties.Creates)

	return Downloaded, nil
}

// extracts path into a new temporary directory in the same directory as path, returns
// the path to the new extracted temp directory
func (w *Watcher) extractAndVerifyToTemp(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty archive path")
	}

	if !iu.FileExist(path) {
		return "", fmt.Errorf("archive file %s does not exist", path)
	}

	parent := filepath.Dir(path)
	if parent == "" {
		return "", fmt.Errorf("invalid temp path")
	}

	td, err := os.MkdirTemp(parent, "choria-archive")
	if err != nil {
		return td, err
	}

	f, err := os.Open(path)
	if err != nil {
		return td, err
	}

	err = w.untar(f, td)
	if err != nil {
		return td, fmt.Errorf("untar failed: %s", err)
	}

	if w.properties.ContentChecksumsChecksum != "" {
		err = w.verify(filepath.Join(td, w.properties.Creates))
		if err != nil {
			w.Errorf("sha256 verify failed: %v", err)
			return td, err
		}
	}

	return td, nil
}

func (w *Watcher) verify(dir string) error {
	ccc, err := w.ProcessTemplate(w.properties.ContentChecksumsChecksum)
	if err != nil {
		return fmt.Errorf("could not parse template on verify_checksum property")
	}
	if ccc == "" {
		return fmt.Errorf("verify_checksum template resulted in an empty string")
	}

	sumsFile := filepath.Join(dir, w.properties.ContentChecksums)
	if !iu.FileIsRegular(sumsFile) {
		return fmt.Errorf("checksums file %s does not exist in the archive (%s)", w.properties.ContentChecksums, sumsFile)
	}

	ok, sum, err := iu.FileHasSha256Sum(sumsFile, ccc)
	if err != nil {
		return fmt.Errorf("failed to checksum file %s: %s", w.properties.ContentChecksums, err)
	}
	if !ok {
		return fmt.Errorf("checksum file %s has an invalid checksum %q != %q", w.properties.ContentChecksums, sum, ccc)
	}

	ok, err = iu.Sha256VerifyDir(sumsFile, dir, nil, func(file string, ok bool) {
		if !ok {
			w.Warnf("Verification checksum failed for %s", file)
		}
	})
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("contents did not pass verification")
	}

	return nil
}

func (w *Watcher) untar(s io.Reader, t string) error {
	uncompressed, err := gzip.NewReader(s)
	if err != nil {
		return fmt.Errorf("unzip failed: %s", err)
	}

	tarReader := tar.NewReader(uncompressed)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("only regular files and directories are supported")
		}

		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("invalid tar file detected")
		}

		path := filepath.Join(t, header.Name)
		if !strings.HasPrefix(path, t) {
			return fmt.Errorf("invalid tar file detected")
		}

		nfo := header.FileInfo()
		if nfo.IsDir() {
			err = os.MkdirAll(path, nfo.Mode())
			if err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, nfo.Mode())
		if err != nil {
			return err
		}
		_, err = io.Copy(file, tarReader)
		file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Watcher) mkTempDir() (string, error) {
	// aagent loader will ignore tmp directory
	parent := filepath.Join(w.properties.TargetDirectory, "tmp")
	if !iu.FileIsDir(parent) {
		err := os.MkdirAll(parent, 0700)
		if err != nil {
			return "", err
		}
	}

	return os.MkdirTemp(parent, "")
}

// creates a temp directory, creates a file in that directory and returns the path to the file
// removes the temp directory on any failure, but leaves it on success - the temp file is in there
func (w *Watcher) downloadSourceToTemp(ctx context.Context) (string, error) {
	source, err := w.ProcessTemplate(w.properties.Source)
	if err != nil {
		return "", fmt.Errorf("source template processing failed: %s", err)
	}

	sourceChecksum, err := w.ProcessTemplate(w.properties.ArchiveChecksum)
	if err != nil {
		return "", fmt.Errorf("checksum template processing failed: %s", err)
	}

	if source == "" {
		return "", fmt.Errorf("source template resulted in an empty string")
	}
	if sourceChecksum == "" {
		return "", fmt.Errorf("checksum template resulted in an empty string")
	}

	w.previousSource = source

	uri, err := url.Parse(source)
	if err != nil {
		return "", fmt.Errorf("invalid url: %s", err)
	}

	td, err := w.mkTempDir()
	if err != nil {
		return "", fmt.Errorf("could not create temp directory: %s", err)
	}
	if td == "" {
		return "", fmt.Errorf("could not create temp directory for unknown reason")
	}

	tf, err := os.CreateTemp(td, "*-archive.tgz")
	if err != nil {
		os.RemoveAll(td)
		return "", fmt.Errorf("could not create temp file: %s", err)
	}
	defer tf.Close()

	w.Infof("Attempting to download %s to %s", uri.String(), tf.Name())

	err = func() error {
		client := http.Client{}

		if w.properties.Insecure {
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
		if err != nil {
			return fmt.Errorf("request failed: %s", err)
		}
		req.Header.Add("User-Agent", fmt.Sprintf("Choria Archive Watcher %s", build.Version))

		if w.properties.Username != "" {
			user, err := w.ProcessTemplate(w.properties.Username)
			if err != nil {
				return fmt.Errorf("invalid username template: %v", err)
			}
			pass, err := w.ProcessTemplate(w.properties.Password)
			if err != nil {
				return fmt.Errorf("invalid password template: %v", err)
			}

			req.SetBasicAuth(user, pass)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %s", err)
		}
		defer resp.Body.Close()

		_, err = io.Copy(tf, resp.Body)
		if err != nil {
			return fmt.Errorf("request failed: %s", err)
		}

		tf.Close()

		ok, sum, err := iu.FileHasSha256Sum(tf.Name(), sourceChecksum)
		if err != nil {
			return fmt.Errorf("archive checksum calculation failed: %s", err)
		}
		if !ok {
			return fmt.Errorf("archive checksum %s != %s missmatch", sum, sourceChecksum)
		}

		return nil
	}()
	if err != nil {
		os.RemoveAll(td)
		return "", err
	}

	if !iu.FileExist(tf.Name()) {
		return "", fmt.Errorf("downloaded file %s does not exist", tf.Name())
	}

	return tf.Name(), nil
}

func (w *Watcher) performWatch(ctx context.Context, force bool) {
	w.wmu.Lock()
	defer w.wmu.Unlock()

	if !force && time.Since(w.lastWatch) < w.interval {
		return
	}

	err := w.handleCheck(w.watch(ctx))
	if err != nil {
		w.Errorf("could not handle watcher event: %s", err)
	}
}

func (w *Watcher) intervalWatcher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	tick := time.NewTicker(w.interval)

	for {
		select {
		case <-tick.C:
			w.performWatch(ctx, false)

		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}

func (w *Watcher) handleCheck(s State, err error) error {
	w.Debugf("handling state for %s %v", stateNames[s], err)

	w.mu.Lock()
	w.previous = s
	w.mu.Unlock()

	switch s {
	case Error:
		if err != nil {
			w.Errorf("Managing archive failed: %s", err)
		}

		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	case Downloaded:
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case VerifiedOK:
		w.NotifyWatcherState(w.CurrentState())
		return w.SuccessTransition()

	case VerifyFailed:
		w.NotifyWatcherState(w.CurrentState())
		return w.FailureTransition()

	}

	return nil
}

func (w *Watcher) setProperties(props map[string]any) error {
	if w.properties == nil {
		w.properties = &Properties{}
	}

	err := util.ParseMapStructure(props, w.properties)
	if err != nil {
		return err
	}

	return w.validate()
}

func (w *Watcher) validate() error {
	if w.properties.Source == "" {
		return fmt.Errorf("source is required")
	}

	if w.properties.Creates == "" {
		return fmt.Errorf("creates is required")
	}

	// TODO need to make sure this is somehow not super dangerous choices like target / creates etc
	// might make this into a machine downloader not a generic downloader
	if w.properties.TargetDirectory == "" {
		return fmt.Errorf("target is required")
	}

	if w.properties.ArchiveChecksum == "" {
		return fmt.Errorf("checksum is required")
	}

	if w.properties.ContentChecksums != "" && w.properties.ContentChecksumsChecksum == "" {
		return fmt.Errorf("verify_checksum is required if verify is set")
	}

	if w.properties.Username != "" && w.properties.Password == "" {
		return fmt.Errorf("password is required when username is given")
	}

	if w.properties.Governor != "" && w.properties.GovernorTimeout == 0 {
		w.Infof("Setting Governor timeout to 5 minutes while unset")
		w.properties.GovernorTimeout = 5 * time.Minute
	}

	if w.properties.Timeout < 5*time.Second {
		w.Infof("Setting timeout to minimum 5 seconds")
		w.properties.Timeout = 5 * time.Second
	}

	return nil
}

func (w *Watcher) CurrentState() any {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := &StateNotification{
		Event:           event.New(w.name, wtype, version, w.machine),
		Source:          w.properties.Source,
		Creates:         w.properties.Creates,
		PreviousOutcome: stateNames[w.previous],
		PreviousRunTime: w.previousRunTime.Nanoseconds(),
	}

	if w.previousSource != "" {
		s.Source = w.previousSource
	}

	return s
}
