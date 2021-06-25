package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"
)

type DirectorySpool struct {
	directory    string
	started      bool
	log          *logrus.Entry
	mu           sync.Mutex
	ctx          context.Context
	cancel       func()
	pollInterval time.Duration
	bo           backoff.Policy
	spoolMax     int
	identity     string
	skipList     map[uint]map[string]struct{} // per priority list of files to skip
}

type Framework interface {
	Configuration() *config.Config
	Logger(component string) *logrus.Entry
}

func NewDirectorySpool(fw Framework) (*DirectorySpool, error) {
	cfg := fw.Configuration()
	d := cfg.Choria.SubmissionSpool
	if d == "" {
		return nil, fmt.Errorf("spool is not configured")
	}

	if cfg.Choria.SubmissionSpoolMaxSize < 1 {
		return nil, fmt.Errorf("spool size is too small")
	}

	if cfg.Identity == "" {
		return nil, fmt.Errorf("identity is unknown")
	}

	spool := &DirectorySpool{
		directory:    d,
		log:          fw.Logger("directory_spool").WithField("directory", d),
		pollInterval: time.Second,
		skipList:     map[uint]map[string]struct{}{},
		bo:           backoff.Default,
		spoolMax:     cfg.Choria.SubmissionSpoolMaxSize,
		identity:     cfg.Identity,
	}

	err := spool.createSpool()
	if err != nil {
		return nil, err
	}

	return spool, nil
}

func (d *DirectorySpool) Discard(m *Message) error {
	if m.st != Directory {
		return fmt.Errorf("not a disk spool message")
	}

	d.log.Debugf("Discarding message %s in %s", m.ID, m.sm)

	d.removeCompletedWithSkip(m.sm.(string), d.skipList[m.Priority])

	return nil
}

func (d *DirectorySpool) Complete(m *Message) error {
	return d.Discard(m)
}

func (d *DirectorySpool) IncrementTries(m *Message) error {
	if m.st != Directory {
		return fmt.Errorf("not a disk spool message")
	}

	m.Tries++
	m.NextTry = time.Now().Add(d.bo.Duration(int(m.Tries)))

	if m.Tries >= m.MaxTries {
		d.log.Debugf("Discarding message %s in %s as it reached max tries %d", m.ID, m.sm, m.MaxTries)
		d.removeCompletedWithSkip(m.sm.(string), d.skipList[m.Priority])
		return nil
	}

	d.log.Debugf("Incrementing message %s in %s tries to %d", m.ID, m.sm, m.Tries)

	jm, err := json.Marshal(m)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(m.sm.(string), jm, 0600)
	if err != nil {
		d.log.Errorf("Could not increment tries in message %s, discarding: %s", m.sm, err)
		d.removeCompletedWithSkip(m.sm.(string), d.skipList[m.Priority])
	}

	return nil
}

func (d *DirectorySpool) removeCompletedWithSkip(path string, skipList map[string]struct{}) {
	if path == "" {
		return
	}

	err := os.Remove(path)
	if err != nil {
		d.log.Errorf("could not remove message: %s: %s", path, err)
		skipList[path] = struct{}{}
	}
}

func (d *DirectorySpool) compactSkipList(skipList map[string]struct{}, known map[string]struct{}) {
	var toRemove []string
	for k := range skipList {
		_, ok := known[k]
		if !ok {
			toRemove = append(toRemove, k)
		}
	}

	for _, r := range toRemove {
		delete(skipList, r)
	}
}

func (d *DirectorySpool) NewMessage() *Message {
	m := newMessage(d.identity)
	m.st = Directory
	m.Identity = d.identity

	return m
}

func (d *DirectorySpool) processDir(priority uint, dir string, handler func([]*Message) error) error {
	skipList := d.skipList[priority]

	entries, err := d.entries(priority)
	if err != nil {
		return err
	}

	var msgs []*Message
	found := make(map[string]struct{})

	for _, entry := range entries {
		found[entry.Name()] = struct{}{}

		msgPath := filepath.Join(dir, entry.Name())
		if _, skip := skipList[msgPath]; skip {
			os.Remove(msgPath) // maybe it works this time
			continue
		}

		msg, err := d.readMessage(filepath.Join(dir, entry.Name()))
		if err != nil {
			d.log.Errorf("could not read message %s, discarding: %s", msgPath, err)
			d.removeCompletedWithSkip(msgPath, skipList)
			continue
		}

		if msg.Tries >= msg.MaxTries {
			d.log.Errorf("removing message %s on try %d/%d", msgPath, msg.Tries, msg.MaxTries)
			d.removeCompletedWithSkip(msgPath, skipList)
			continue
		}

		if msg.Tries > 0 && time.Now().Before(msg.NextTry) {
			d.log.Errorf("skipping message %s till %v", msgPath, msg.NextTry)
			continue
		}

		msg.sm = msgPath
		msgs = append(msgs, msg)
	}

	// dont leak skiplist entries for ones that somehow got deleted
	d.compactSkipList(skipList, found)

	if len(msgs) == 0 {
		return nil
	}

	return handler(msgs)
}

func (d *DirectorySpool) readMessage(path string) (*Message, error) {
	jmsg, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var msg Message
	err = json.Unmarshal(jmsg, &msg)
	if err != nil {
		return nil, err
	}
	msg.st = Directory
	msg.sm = path

	return &msg, nil
}

func (d *DirectorySpool) entries(priority uint) ([]fs.FileInfo, error) {
	found, err := ioutil.ReadDir(filepath.Join(d.directory, fmt.Sprintf("P%d", priority)))
	if err != nil {
		return nil, err
	}

	sort.Slice(found, func(i, j int) bool {
		return found[i].Name() < found[j].Name()
	})

	var entries []fs.FileInfo
	for _, f := range found {
		if strings.HasSuffix(f.Name(), ".msg") {
			entries = append(entries, f)
		}
	}

	return entries, nil
}

func (d *DirectorySpool) countEntries(priority uint) int {
	entries, err := d.entries(priority)
	if err != nil {
		return 0
	}

	return len(entries)
}

func (d *DirectorySpool) worker(ctx context.Context, wg *sync.WaitGroup, priority uint, handler func([]*Message) error, ready chan struct{}) {
	defer wg.Done()

	pollTick := time.NewTicker(d.pollInterval)
	dir := filepath.Join(d.directory, fmt.Sprintf("P%d", priority))
	var mu sync.Mutex

	d.log.Infof("Worker %d starting on %s", priority, dir)

	// files we cant remove, cant write to or just generally cause hassles will
	// be added here for a ephemeral list of files just to ignore
	d.skipList[priority] = map[string]struct{}{}

	ready <- struct{}{}

	for {
		select {
		case <-pollTick.C:
			mu.Lock()
			err := d.processDir(priority, dir, handler)
			mu.Unlock()
			if err != nil {
				d.log.Errorf("Polling %s failed: %s", dir, err)
			}

		case <-ctx.Done():
			pollTick.Stop()
			d.log.Debugf("Spool worker P%d exiting: %s", priority, ctx.Err())
			return
		}
	}
}
func (d *DirectorySpool) StartPoll(ctx context.Context, wg *sync.WaitGroup, handler func([]*Message) error) error {
	defer wg.Done()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.started {
		return fmt.Errorf("already started")
	}

	d.started = true

	d.ctx, d.cancel = context.WithCancel(ctx)

	ready := make(chan struct{}, 1)

	for p := uint(0); p < 5; p++ {
		wg.Add(1)
		go d.worker(d.ctx, wg, p, handler, ready)
		<-ready
	}

	return nil
}

func (d *DirectorySpool) newMsgPath(priority uint) string {
	return filepath.Join(d.directory, fmt.Sprintf("P%d", priority), fmt.Sprintf("%d-%s.msg", time.Now().UnixNano(), util.UniqueID()))
}

func (d *DirectorySpool) Submit(msg *Message) error {
	err := msg.Validate()
	if err != nil {
		return err
	}

	cnt := d.countEntries(msg.Priority)
	if cnt > d.spoolMax {
		return fmt.Errorf("spool is full")
	}

	j, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	f, err := ioutil.TempFile(d.directory, "")
	if err != nil {
		return err
	}
	f.Close()
	defer os.Remove(f.Name())

	err = ioutil.WriteFile(f.Name(), j, 0600)
	if err != nil {
		return err
	}

	out := d.newMsgPath(msg.Priority)
	err = os.Rename(f.Name(), out)
	if err != nil {
		return err
	}

	d.log.Infof("Submitted a message to %s", out)

	return nil
}

func (d *DirectorySpool) createSpool() error {
	for i := 0; i < 5; i++ {
		err := os.MkdirAll(filepath.Join(d.directory, fmt.Sprintf("P%d", i)), 0700)
		if err != nil {
			return err
		}
	}

	return nil
}
