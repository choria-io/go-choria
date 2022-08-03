// Copyright (c) 2021-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/shlex"

	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/protocol"

	"github.com/sirupsen/logrus"
)

// External implements discovery via externally executed binaries
type External struct {
	fw      client.ChoriaFramework
	timeout time.Duration
	log     *logrus.Entry
}

// Response is the expected response from the external script on its STDOUT
type Response struct {
	Protocol string   `json:"protocol"`
	Nodes    []string `json:"nodes"`
	Error    string   `json:"error"`
}

// Request is the request sent to the external script on its STDIN
type Request struct {
	Protocol   string            `json:"protocol"`
	Collective string            `json:"collective"`
	Filter     *protocol.Filter  `json:"filter"`
	Options    map[string]string `json:"options"`
	Schema     string            `json:"$schema"`
	Timeout    float64           `json:"timeout"`
}

const (
	// ResponseProtocol is the protocol responses from the external script should have
	ResponseProtocol = "io.choria.choria.discovery.v1.external_reply"
	// RequestProtocol is a protocol set in the request that the external script can validate
	RequestProtocol = "io.choria.choria.discovery.v1.external_request"
	// RequestSchema is the location to a JSON Schema for requests
	RequestSchema = "https://choria.io/schemas/choria/discovery/v1/external_request.json"
)

func New(fw client.ChoriaFramework) *External {
	return &External{
		fw:      fw,
		timeout: time.Second * time.Duration(fw.Configuration().DiscoveryTimeout),
		log:     fw.Logger("external_discovery"),
	}
}

func (e *External) Discover(ctx context.Context, opts ...DiscoverOption) (n []string, err error) {
	dopts := &dOpts{
		collective: e.fw.Configuration().MainCollective,
		timeout:    e.timeout,
		command:    e.fw.Configuration().Choria.ExternalDiscoveryCommand,
		do:         make(map[string]string),
	}

	for _, opt := range opts {
		opt(dopts)
	}

	if dopts.filter == nil {
		dopts.filter = protocol.NewFilter()
	}

	if dopts.timeout < time.Second {
		e.log.Warnf("Forcing discovery timeout to minimum 1 second")
		dopts.timeout = time.Second
	}

	command, ok := dopts.do["command"]
	if ok && command != "" {
		dopts.command = command
		delete(dopts.do, "command")
	}

	if dopts.command == "" {
		return nil, fmt.Errorf("no command specified for external discovery")
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, dopts.timeout)
	defer cancel()

	idat := &Request{
		Schema:     RequestSchema,
		Protocol:   RequestProtocol,
		Timeout:    dopts.timeout.Seconds(),
		Collective: dopts.collective,
		Filter:     dopts.filter,
		Options:    dopts.do,
	}

	req, err := json.Marshal(idat)
	if err != nil {
		return nil, fmt.Errorf("could not encode the filter: %s", err)
	}

	reqfile, err := os.CreateTemp("", "request")
	if err != nil {
		return nil, fmt.Errorf("could not create request temp file: %s", err)
	}
	defer os.Remove(reqfile.Name())

	repfile, err := os.CreateTemp("", "reply")
	if err != nil {
		return nil, fmt.Errorf("could not create reply temp file: %s", err)
	}
	defer os.Remove(repfile.Name())
	repfile.Close()

	_, err = reqfile.Write(req)
	if err != nil {
		return nil, fmt.Errorf("could not create reply temp file: %s", err)
	}

	var args []string
	parts, err := shlex.Split(dopts.command)
	if err != nil {
		return nil, err
	}
	args = append(args, parts[0])
	if len(parts) > 1 {
		args = append(args, parts[1:]...)
	}
	args = append(args, reqfile.Name(), repfile.Name(), RequestProtocol)
	command = args[0]

	cmd := exec.CommandContext(timeoutCtx, command, args[1:]...)
	cmd.Dir = os.TempDir()
	cmd.Env = []string{
		"CHORIA_EXTERNAL_REQUEST=" + reqfile.Name(),
		"CHORIA_EXTERNAL_REPLY=" + repfile.Name(),
		"CHORIA_EXTERNAL_PROTOCOL=" + RequestProtocol,
		"PATH=" + os.Getenv("PATH"),
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open STDOUT: %s", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open STDERR: %s", err)
	}

	wg := &sync.WaitGroup{}
	outputReader := func(wg *sync.WaitGroup, in io.ReadCloser, logger func(args ...any)) {
		defer wg.Done()

		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			logger(scanner.Text())
		}
	}

	wg.Add(1)
	go outputReader(wg, stderr, e.log.Error)
	wg.Add(1)
	go outputReader(wg, stdout, e.log.Info)

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("executing %s failed: %s", filepath.Base(command), err)
	}

	cmd.Wait()
	wg.Wait()

	if cmd.ProcessState.ExitCode() != 0 {
		return nil, fmt.Errorf("executing %s failed: exit status %d", filepath.Base(command), cmd.ProcessState.ExitCode())
	}

	repjson, err := os.ReadFile(repfile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read reply json: %s", err)
	}

	var resp Response
	err = json.Unmarshal(repjson, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode reply json: %s", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	if resp.Protocol != ResponseProtocol {
		return nil, fmt.Errorf("invalid response received, expected protocol %q got %q", ResponseProtocol, resp.Protocol)
	}

	return resp.Nodes, nil
}
