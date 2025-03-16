// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/inter"
	iu "github.com/choria-io/go-choria/internal/util"
	"github.com/choria-io/go-choria/submission"
	"github.com/sirupsen/logrus"
)

// Process describes a process managed by the execution provider
type Process struct {
	Action        string            `json:"action"`
	Agent         string            `json:"agent"`
	Args          []string          `json:"args"`
	Caller        string            `json:"caller"`
	Created       time.Time         `json:"created"`
	Command       string            `json:"command"`
	Environment   map[string]string `json:"environment"`
	HeartBeat     time.Duration     `json:"heartbeat"`
	ID            string            `json:"id"`
	Identity      string            `json:"identity"`
	PidFile       string            `json:"pid"`
	RequestID     string            `json:"requestid"`
	StartTime     time.Time         `json:"start,omitempty"`
	StderrFile    string            `json:"stderr"`
	StdoutFile    string            `json:"stdout"`
	TerminateTime time.Time         `json:"terminate,omitempty"`
}

// ListQuery configures the job lister
type ListQuery struct {
	Action    string    `json:"action"`
	Agent     string    `json:"agent"`
	Before    time.Time `json:"before"`
	Caller    string    `json:"caller"`
	Command   string    `json:"command"`
	Completed bool      `json:"completed"`
	Identity  string    `json:"identity"`
	RequestID string    `json:"requestid"`
	Running   bool      `json:"running"`
	Since     time.Time `json:"since"`
}

type ExitCode struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

type Submitter interface {
	NewMessage() *submission.Message
	Submit(msg *submission.Message) error
}

type watchedLine struct {
	origin string
	line   []byte
}

var (
	ErrSpoolNotConfigured     = errors.New("spool not configured")
	ErrSpoolNotFound          = errors.New("spool not found")
	ErrSpecificationNotFound  = errors.New("specification not found")
	ErrSpecificationLoadError = errors.New("specification could not be loaded")
	ErrDuplicateJob           = errors.New("duplicate job")
	ErrStartFailed            = errors.New("start failed")
	ErrInvalidProcess         = errors.New("invalid process")
	ErrWritingPidFailed       = errors.New("writing pid file failed")
	ErrInvalidPid             = errors.New("invalid pid file")
	ErrAlreadyStarted         = errors.New("already started")
	ErrSpoolCreationFailed    = errors.New("spool creation failed")
	ErrProcessFailed          = errors.New("process failed")
	ErrQueryRequired          = errors.New("query required")
)

func New(caller string, agent string, action string, reqID string, identity string, id string, command string, args []string, env map[string]string) (*Process, error) {
	if caller == "" {
		return nil, fmt.Errorf("%w: no caller", ErrInvalidProcess)
	}
	if reqID == "" {
		return nil, fmt.Errorf("%w: no request id", ErrInvalidProcess)
	}
	if id == "" {
		return nil, fmt.Errorf("%w: no id", ErrInvalidProcess)
	}
	if identity == "" {
		return nil, fmt.Errorf("%w: no identity", ErrInvalidProcess)
	}
	if command == "" {
		return nil, fmt.Errorf("%w: no command", ErrInvalidProcess)
	}

	return &Process{
		Command:     command,
		Args:        args,
		Environment: env,
		Identity:    identity,
		ID:          id,
		Caller:      caller,
		RequestID:   reqID,
		Agent:       agent,
		Action:      action,
		Created:     time.Now().UTC(),
	}, nil
}

func List(spool string, q *ListQuery) ([]*Process, error) {
	if spool == "" {
		return nil, ErrSpoolNotConfigured
	}
	if q == nil {
		return nil, ErrQueryRequired
	}

	result := make([]*Process, 0)

	err := filepath.WalkDir(spool, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.Type().IsRegular() {
			return nil
		}

		proc, err := Load(spool, d.Name())
		if err != nil {
			return nil
		}

		if proc.IsMatch(q) {
			result = append(result, proc)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ListWithChoria(fw inter.Framework, q *ListQuery) ([]*Process, error) {
	return List(fw.Configuration().Choria.ExecutorSpool, q)
}

func Load(spool string, id string) (*Process, error) {
	jobSpec := specPath(spool, id)

	if spool == "" {
		return nil, ErrSpoolNotConfigured
	}

	if !iu.FileIsDir(spool) {
		return nil, ErrSpoolNotFound
	}

	if !iu.FileExist(jobSpec) {
		return nil, ErrSpecificationNotFound
	}

	j, err := os.ReadFile(jobSpec)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSpecificationLoadError, err)
	}

	var p Process
	err = json.Unmarshal(j, &p)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSpecificationLoadError, err)
	}

	return &p, nil
}

func LoadWithChoria(fw inter.Framework, id string) (*Process, error) {
	return Load(fw.Configuration().Choria.ExecutorSpool, id)
}

// IsMatch determines if the processes matches the query
func (p *Process) IsMatch(q *ListQuery) bool {
	matcher := func(matched []bool, should bool, property bool) []bool {
		if !should {
			return matched
		}

		matched = append(matched, property)

		return matched
	}
	started, _ := p.HasStarted()

	matched := matcher(nil, q.Running, p.IsRunning())
	matched = matcher(matched, q.Completed, started && !p.IsRunning())
	matched = matcher(matched, q.Caller != "", p.Caller == q.Caller)
	matched = matcher(matched, q.Agent != "", p.Agent == q.Agent)
	matched = matcher(matched, q.Action != "", p.Action == q.Action)
	matched = matcher(matched, q.RequestID != "", p.RequestID == q.RequestID)
	matched = matcher(matched, q.Command != "", p.Command == q.Command)
	matched = matcher(matched, q.Identity != "", p.Identity == q.Identity)
	matched = matcher(matched, !q.Since.IsZero(), p.Created.After(q.Since))
	matched = matcher(matched, !q.Before.IsZero(), p.Created.Before(q.Before))

	return len(matched) > 0 && !slices.Contains(matched, false)
}

// CreateSpool creates the spool and saves the spec, fails if already created
func (p *Process) CreateSpool(spool string) (json.RawMessage, error) {
	if spool == "" {
		return nil, ErrSpoolNotConfigured
	}

	has, err := hasJob(spool, p.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidProcess, err)
	}
	if has {
		return nil, fmt.Errorf("%w: spool already exists", ErrDuplicateJob)
	}

	jobDir := filepath.Join(spool, p.ID)
	err = os.Mkdir(jobDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSpoolCreationFailed, err)
	}

	return saveJobSpec(spool, p)
}

// StartSupervised starts a process attached to the calling process with status, heartbeats and optionally output published to Choria Submission
func (p *Process) StartSupervised(ctx context.Context, spool string, submit Submitter, heartbeat time.Duration, publishOutput bool, log *logrus.Entry) error {
	if spool == "" {
		return ErrSpoolNotConfigured
	}

	if !p.StartTime.IsZero() {
		return ErrAlreadyStarted
	}

	log = log.WithFields(logrus.Fields{
		"id":      p.ID,
		"command": p.Command,
		"caller":  p.Caller,
		"request": p.RequestID,
	})

	log.Infof("Starting supervised process")

	p.StdoutFile = stdOutPath(spool, p.ID)
	p.StderrFile = stdErrPath(spool, p.ID)
	p.PidFile = pidPath(spool, p.ID)
	p.StartTime = time.Now().UTC()
	p.HeartBeat = heartbeat

	prefix := fmt.Sprintf("choria.executor.%s.%s", p.RequestID, p.ID)

	jProc, err := saveJobSpec(spool, p)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStartFailed, err)
	}

	msg := newSubmissionMessage(submit, fmt.Sprintf("%s.spec", prefix))
	msg.Payload = jProc
	err = submit.Submit(msg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStartFailed, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg := &sync.WaitGroup{}

	var closers []io.Closer
	var stdout, stderr io.WriteCloser

	if publishOutput {
		var stdoutReader, stderrReader *io.PipeReader
		var stdoutFile, stderrFile *os.File

		stdoutFile, err = os.Create(p.StdoutFile)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrStartFailed, err)
		}
		stderrFile, err = os.Create(p.StderrFile)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrStartFailed, err)
		}

		stdoutReader, stdout = io.Pipe()
		stderrReader, stderr = io.Pipe()

		closers = append(closers, stdout, stderr, stdoutFile, stderrFile, stdoutReader, stderrReader)

		wg.Add(1)
		go watchOutput(wg, io.TeeReader(stdoutReader, stdoutFile), io.TeeReader(stderrReader, stderrFile), submit, prefix, p, log)
	} else {
		stdout, err = os.Create(p.StdoutFile)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrStartFailed, err)
		}
		stderr, err = os.Create(p.StderrFile)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrStartFailed, err)
		}

		closers = append(closers, stdout, stderr)
	}

	env := createEnv(p.Environment)
	cmd := exec.CommandContext(ctx, p.Command, p.Args...)
	cmd.Dir = "/"
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStartFailed, err)
	}

	// this could fail and the command could be running already...
	err = os.WriteFile(p.PidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0700)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrWritingPidFailed, err)
	}

	if heartbeat > 0 {
		wg.Add(1)
		go hb(ctx, wg, heartbeat, submit, prefix, p, log)
	}

	msg = newSubmissionMessage(submit, fmt.Sprintf("%s.pid", prefix))
	msg.Payload = []byte(strconv.Itoa(cmd.Process.Pid))
	err = submit.Submit(msg)
	if err != nil {
		log.Errorf("Failed to publish start update: %v", err)
	}

	var state *os.ProcessState
	var waitErr error

	if cmd.Process == nil {
		waitErr = ErrStartFailed
	} else {
		state, waitErr = cmd.Process.Wait()
	}

	// always save then handle the error from waiting
	p.TerminateTime = time.Now().UTC()
	_, saveErr := saveJobSpec(spool, p)
	if saveErr != nil {
		log.Errorf("Could not save job after execution: %v", err)
	}

	cancel()
	for _, closer := range closers {
		closer.Close()
	}

	wg.Wait()

	msg = newSubmissionMessage(submit, fmt.Sprintf("%s.exit", prefix))
	errMsg := ""
	if waitErr != nil {
		errMsg = waitErr.Error()
	}
	if state == nil {
		msg.Payload = exitJson(-1, errMsg)
	} else {
		exitCode := state.ExitCode()
		if exitCode == -1 {
			errMsg = "process killed"
		}
		msg.Payload = exitJson(exitCode, errMsg)
	}
	err = submit.Submit(msg)
	if err != nil {
		log.Errorf("Failed to publish exit update: %v", err)
	}

	err = os.WriteFile(exitPath(spool, p.ID), msg.Payload, 0700)
	if err != nil {
		log.Errorf("Failed to write exit code: %v", saveErr)
	}

	if waitErr != nil {
		log.Errorf("Failed to wait for process to exit: %v", waitErr)
		return nil
	}

	log.Infof("Finished supervised process")

	return nil
}

// HasStarted determines if the command was started by the presence of the PID file
func (p *Process) HasStarted() (bool, error) {
	if p.PidFile == "" {
		return false, nil
	}

	stat, err := os.Stat(p.PidFile)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return !stat.IsDir(), nil
}

// IsRunning checks if the process is running
func (p *Process) IsRunning() bool {
	if runtime.GOOS != "windows" {
		err := p.Signal(syscall.Signal(0))

		return err == nil
	}

	pid, err := p.ParsePid()
	if err != nil {
		return false
	}

	_, err = os.FindProcess(pid)
	return err == nil
}

// Signal sends a signal to the process
func (p *Process) Signal(sig syscall.Signal) error {
	pid, err := p.ParsePid()
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return proc.Signal(sig)
}

// Stderr reads the stderr output
func (p *Process) Stderr() ([]byte, error) {
	if p.StderrFile == "" {
		return nil, fmt.Errorf("%w: no stderr file configured", ErrInvalidProcess)
	}

	return os.ReadFile(p.StderrFile)
}

// Stdout reads the stdout output
func (p *Process) Stdout() ([]byte, error) {
	if p.StdoutFile == "" {
		return nil, fmt.Errorf("%w: no stdout file configured", ErrInvalidProcess)
	}

	return os.ReadFile(p.StdoutFile)
}

// ParsePid loads and parses the pid file, returns -1 on error
func (p *Process) ParsePid() (int, error) {
	if p.PidFile == "" {
		return -1, fmt.Errorf("%w: no pid file configured", ErrInvalidProcess)
	}

	pidBytes, err := os.ReadFile(p.PidFile)
	if err != nil {
		return -1, fmt.Errorf("%w: %w", ErrInvalidPid, err)
	}

	if len(pidBytes) == 0 {
		return -1, fmt.Errorf("%w: 0 length pid", ErrInvalidPid)
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return -1, fmt.Errorf("%w: %w", ErrInvalidPid, err)
	}

	if pid == 1 {
		return -1, fmt.Errorf("%w: impossible pid", ErrInvalidPid)
	}

	return pid, nil
}

// ParseExitCode parse the exist code file, -1 on error
func (p *Process) ParseExitCode() (int, error) {
	started, err := p.HasStarted()
	if err != nil {
		return -1, err
	}
	if !started {
		return -1, fmt.Errorf("%w: process not started", ErrInvalidProcess)
	}

	exitBytes, err := os.ReadFile(exitPath(filepath.Dir(filepath.Dir(p.PidFile)), p.ID))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidProcess, err)
	}

	var exit ExitCode
	err = json.Unmarshal(exitBytes, &exit)
	if err != nil {
		return -1, fmt.Errorf("%w: %w", ErrInvalidProcess, err)
	}

	if exit.Error != "" {
		return -1, fmt.Errorf("%w: %w", ErrProcessFailed, errors.New(exit.Error))
	}

	return exit.Code, nil
}

func watchOutputReader(wg *sync.WaitGroup, r io.Reader, origin string, out chan watchedLine, log *logrus.Entry) {
	defer wg.Done()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		out <- watchedLine{origin: origin, line: scanner.Bytes()}
	}
}

func hb(ctx context.Context, wg *sync.WaitGroup, heartbeat time.Duration, submit Submitter, prefix string, p *Process, log *logrus.Entry) {
	defer wg.Done()

	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			isRunning := p.IsRunning()
			msg := newSubmissionMessage(submit, fmt.Sprintf("%s.hb", prefix))
			msg.Payload = []byte(fmt.Sprintf("%t", isRunning))

			err := submit.Submit(msg)
			if err != nil {
				log.Errorf("Failed to publish heartbeat: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func createEnv(env map[string]string) []string {
	tenv := map[string]string{
		"PATH": os.Getenv("PATH"),
	}

	for k, v := range env {
		tenv[k] = v
	}

	var res []string
	for k, v := range tenv {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}

	return res
}

func watchOutput(wg *sync.WaitGroup, stdout io.Reader, stderr io.Reader, submit Submitter, prefix string, p *Process, log *logrus.Entry) {
	defer wg.Done()

	lines := make(chan watchedLine, 10)

	owg := &sync.WaitGroup{}
	owg.Add(2)
	go watchOutputReader(owg, stderr, "stderr", lines, log)
	go watchOutputReader(owg, stdout, "stdout", lines, log)

	// closing the files will stop the watcher routines which
	// this one is waiting on, when thats done it closes the
	// channel which stops the range below
	go func() {
		owg.Wait()
		close(lines)
	}()

	publish := func(prev string, buff *bytes.Buffer) {
		if buff.Len() == 0 {
			return
		}

		msg := newSubmissionMessage(submit, "")

		switch prev {
		case "stdout":
			msg.Subject = fmt.Sprintf("%s.out.stdout", prefix)
		case "stderr":
			msg.Subject = fmt.Sprintf("%s.out.stderr", prefix)
		}
		msg.Payload = buff.Bytes()
		err := submit.Submit(msg)
		if err != nil {
			log.Errorf("Failed to publish log update: %v", err)
		} else {
			buff.Reset()
		}
	}

	ticker := time.NewTicker(time.Second)
	buff := bytes.NewBuffer([]byte{})
	prev := ""

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				publish(prev, buff)
				return
			}

			if prev == "" {
				prev = line.origin
			}
			if line.origin != prev {
				publish(prev, buff)
			}
			if buff.Len() > 1024 {
				publish(prev, buff)
			}
			prev = line.origin

			buff.Write(line.line)
			buff.Write([]byte("\n"))

		case <-ticker.C:
			publish(prev, buff)
		}
	}
}

func saveJobSpec(spool string, proc *Process) (json.RawMessage, error) {
	j, err := json.Marshal(proc)
	if err != nil {
		return nil, err
	}

	tf, err := os.CreateTemp(spool, "")
	if err != nil {
		return nil, err
	}

	err = os.Chmod(tf.Name(), 0700)
	if err != nil {
		tf.Close()
		os.Remove(tf.Name())

		return nil, err
	}

	_, err = tf.Write(j)
	if err != nil {
		tf.Close()
		os.Remove(tf.Name())
		return nil, err
	}
	tf.Close()

	return j, os.Rename(tf.Name(), specPath(spool, proc.ID))
}

func hasJob(spool string, id string) (bool, error) {
	return iu.FileExist(specPath(spool, id)), nil
}

func pidPath(spool string, id string) string {
	return filepath.Join(spool, id, "pid")
}

func stdOutPath(spool string, id string) string {
	return filepath.Join(spool, id, "stdout")
}

func stdErrPath(spool string, id string) string {
	return filepath.Join(spool, id, "stderr")
}

func specPath(spool string, id string) string {
	return filepath.Join(spool, id, "spec.json")
}

func exitPath(spool string, id string) string {
	return filepath.Join(spool, id, "exit")
}

func exitJson(exitCode int, err string) json.RawMessage {
	return []byte(fmt.Sprintf(`{"code":%d,"error":%q}`, exitCode, err))
}

func newSubmissionMessage(submit Submitter, subject string) *submission.Message {
	msg := submit.NewMessage()
	msg.Subject = subject
	msg.Priority = 1
	msg.Reliable = true

	return msg
}
