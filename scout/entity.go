package scout

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/logger"
	"github.com/choria-io/go-choria/scout/check"
	"github.com/choria-io/go-choria/scout/overrides"
	"github.com/choria-io/go-choria/scout/tags"
)

type Entity struct {
	id         string
	nc         *nats.Conn
	machineDir string

	overridesFile string
	tagCheckMgr   map[string]*tags.Manager
	overridesMgr  *overrides.Manager
	checkMgrs     map[string]*check.Manager
	checkLists    map[string][]string
	fw            Framework
	ctx           context.Context
	wg            *sync.WaitGroup

	logger.Logrus
	sync.Mutex
}

type Framework interface {
	Identity() string
	NATSConn() *nats.Conn
	Logger(string) *logrus.Entry
	OverridesFile() string
	Tags() ([]string, error)
	MachineSourceDir() string
}

func NewEntity(ctx context.Context, wg *sync.WaitGroup, fw Framework) (*Entity, error) {
	var err error

	e := &Entity{
		id:            fw.Identity(),
		nc:            fw.NATSConn(),
		machineDir:    fw.MachineSourceDir(),
		overridesFile: fw.OverridesFile(),
		fw:            fw,
		ctx:           ctx,
		wg:            wg,
		tagCheckMgr:   make(map[string]*tags.Manager),
		checkMgrs:     make(map[string]*check.Manager),
		checkLists:    make(map[string][]string),
		Logrus:        fw.Logger("entity"),
	}

	etags, err := fw.Tags()
	if err != nil {
		return nil, err
	}

	for _, tag := range etags {
		checks, err := tags.NewManager(tag, fw)
		if err != nil {
			return nil, err
		}

		e.tagCheckMgr[tag] = checks
	}

	e.overridesMgr, err = overrides.NewManager(fw)
	if err != nil {
		return nil, err
	}

	err = e.tagListen(ctx, wg)
	if err != nil {
		return nil, err
	}

	err = e.overrideListen(ctx, wg)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Entity) stringInSet(n string, stack []string) bool {
	for _, i := range stack {
		if n == i {
			return true
		}
	}

	return false
}

func (e *Entity) reconcileChecks() error {
	var running []string
	checks := e.checkNamesUnlocked()

	// remove checks that shouldnt exist anymore
	for k, c := range e.checkMgrs {
		if !e.stringInSet(k, checks) {
			c.Stop(true)
			continue
		}

		running = append(running, k)
	}

	// start checks that isnt in the running set
	for _, c := range checks {
		if !e.stringInSet(c, running) {

			mgr, err := check.NewCheckManager(c, e.fw)
			if err != nil {
				e.Errorf("Could not start check manager %s: %s", c, err)
				continue
			}
			err = mgr.Start(e.ctx, e.wg)
			if err != nil {
				e.Errorf("Could not start check manager %s: %s", c, err)
				continue
			}

			e.checkMgrs[c] = mgr
			e.Infof("Started check %s", c)
		}
	}

	return nil
}

func (e *Entity) overrideListen(ctx context.Context, wg *sync.WaitGroup) error {
	e.Lock()
	defer e.Unlock()

	e.overridesMgr.Subscribe(e.handleOverrideUpdates)
	err := e.overridesMgr.Start(ctx, wg)
	if err != nil {
		return err
	}

	return nil
}

func (e *Entity) tagListen(ctx context.Context, wg *sync.WaitGroup) error {
	e.Lock()
	defer e.Unlock()

	var err error

	for tag, check := range e.tagCheckMgr {
		check.Subscribe(e.handleTagUpdates(tag))
		err = check.Start(ctx, wg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) handleOverrideUpdates() {
	e.Lock()
	defer e.Unlock()

	data, err := e.overridesMgr.JSON()
	if err != nil {
		e.Errorf("Could not update overrides data: %s", err)
		return
	}

	if len(data) == 0 {
		data = []byte(`{}`)
	}

	tf, err := ioutil.TempFile("", "")
	if err != nil {
		e.Errorf("Could not open temporary file for overrides update: %s", err)
		return
	}
	tf.Close()

	err = ioutil.WriteFile(tf.Name(), data, 0644)
	if err != nil {
		e.Errorf("Could not write temporary override data: %s", err)
		return
	}

	err = os.Rename(tf.Name(), e.overridesFile)
	if err != nil {
		e.Errorf("Could not rename tempoary file to %s: %s", e.overridesFile, err)
		return
	}

	e.Infof("Updated check overrides in %s", e.overridesFile)
}

func (e *Entity) handleTagUpdates(tag string) func() {
	return func() {
		e.Infof("Handling tag update for %s", tag)
		e.Lock()
		defer e.Unlock()

		e.checkLists[tag] = e.tagCheckMgr[tag].Checks()
		e.Infof("Starting tag reconcile with checklist: %v", e.checkLists[tag])
		err := e.reconcileChecks()
		if err != nil {
			e.Errorf("Check reconciliation failed: %s", err)
		}
		e.Infof("Completed tag reconcile")
	}
}

func (e *Entity) checkNames() []string {
	e.Lock()
	checks := e.checkNamesUnlocked()
	e.Unlock()

	sort.Strings(checks)

	return checks
}

// gets a unique list of check names from all the lists
func (e *Entity) checkNamesUnlocked() []string {
	checks := make(map[string]struct{})
	for _, list := range e.checkLists {
		for _, item := range list {
			checks[item] = struct{}{}
		}

	}

	var names []string
	for k := range checks {
		names = append(names, k)
	}

	sort.Strings(names)

	return names
}
