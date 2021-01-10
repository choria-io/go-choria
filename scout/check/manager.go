package check

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/machine"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/logger"
	"github.com/choria-io/go-choria/scout/stream"
	"github.com/choria-io/go-choria/scout/updatenotifier"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	check      *Check
	name       string
	subj       string
	mgr        *stream.Mgr
	nc         *nats.Conn
	machineDir string
	ctx        context.Context
	cancel     func()

	logger.Logrus
	updatenotifier.Notifier
	sync.Mutex
}

type Framework interface {
	NATSConn() *nats.Conn
	Logger(string) *logrus.Entry
	MachineSourceDir() string
}

func NewCheckManager(name string, fw Framework) (*Manager, error) {
	c := &Manager{
		check:      &Check{},
		name:       name,
		nc:         fw.NATSConn(),
		machineDir: filepath.Join(fw.MachineSourceDir(), name),
		subj:       "scout.check." + name,
		Logrus:     fw.Logger("checks").WithField("check", name),
	}

	mgr, err := stream.New("scout_check_"+name, c.subj, fw)
	if err != nil {
		return nil, err
	}

	c.mgr = mgr

	return c, nil
}

func (c *Manager) Start(ctx context.Context, wg *sync.WaitGroup) error {
	c.Lock()
	defer c.Unlock()

	c.Infof("Listening for check updates")

	c.ctx, c.cancel = context.WithCancel(ctx)

	wg.Add(1)
	go c.Notifier.Update(c.ctx, wg)

	return c.mgr.Manage(c)
}

func (c *Manager) Stop(rm bool) {
	c.Lock()
	defer c.Unlock()

	if rm {
		err := os.RemoveAll(c.machineDir)
		if err != nil {
			c.Errorf("Could not remove %s: %s", c.machineDir, err)
		}
	}

	c.Infof("Stopping check %s in %s", c.name, c.machineDir)

	if c.cancel != nil {
		c.cancel()
	}
}

// Instance implements stream.updatable
func (c *Manager) Instance() interface{} {
	return &Check{}
}

// Update implements stream.updatable
func (c *Manager) Update(u interface{}) {
	c.Debugf("received check notification %v", u)

	update, ok := u.(*Check)
	if !ok {
		return
	}

	if update.Name != c.name {
		c.Errorf("Received a check update for %s, discarding", update.Name)
		return
	}

	if update.Plugin == "" && update.Builtin == "" {
		c.Errorf("Invalid check update: both plugin and builtin are empty, discarding")
		return
	}

	if update.Plugin != "" && update.Builtin != "" {
		c.Errorf("Invalid check update: both plugin and builtin are set, discarding")
	}

	if update.CheckInterval == 0 {
		update.CheckInterval = 5 * time.Minute
	}

	if update.PluginTimeout == 0 {
		update.PluginTimeout = 10 * time.Second
	}

	if update.RemediateInterval == 0 {
		update.RemediateInterval = 15 * time.Minute
	}

	c.Lock()
	c.check = update

	err := c.WriteMachine()
	if err != nil {
		c.Errorf("Could not write machine: %s", err)
	}

	c.Unlock()

	c.Infof("Received a check update for %s in %s", c.name, c.machineDir)

	c.Notify()
}

func (c *Manager) WriteMachine() error {
	cs := []string{"UNKNOWN", "OK", "WARNING", "CRITICAL", "FORCE_CHECK"}
	check := machine.Machine{
		MachineName:    c.name,
		MachineVersion: "1.0.0",
		InitialState:   "UNKNOWN",
		SplayStart:     0,
		Transitions: []*machine.Transition{
			{Name: "UNKNOWN", From: cs, Destination: "UNKNOWN"},
			{Name: "OK", From: cs, Destination: "OK"},
			{Name: "WARNING", From: cs, Destination: "WARNING"},
			{Name: "CRITICAL", From: cs, Destination: "CRITICAL"},
			{Name: "FORCE_CHECK", From: cs, Destination: "FORCE_CHECK"},
			{Name: "MAINTENANCE", From: cs, Destination: "MAINTENANCE"},
			{Name: "RESUME", From: []string{"MAINTENANCE"}, Destination: "FORCE_CHECK"},
		},
		WatcherDefs: []*watchers.WatcherDef{},
	}

	checkDef := &watchers.WatcherDef{
		Name:       c.name,
		Type:       "nagios",
		StateMatch: []string{"UNKNOWN", "OK", "CRITICAL", "CRITICAL"},
		Interval:   c.check.CheckInterval.String(),
		Properties: map[string]interface{}{
			"timeout": c.check.PluginTimeout.String(),
		},
	}

	switch {
	case c.check.Builtin != "":
		checkDef.Properties["builtin"] = c.check.Builtin

	case c.check.Plugin != "":
		if len(c.check.Arguments) > 0 {
			checkDef.Properties["plugin"] = fmt.Sprintf("%s %s", c.check.Plugin, c.check.Arguments)
		} else {
			checkDef.Properties["plugin"] = c.check.Plugin
		}
	}

	check.WatcherDefs = append(check.WatcherDefs, checkDef)

	if c.check.RemediateCommand != "" {
		check.WatcherDefs = append(check.WatcherDefs, &watchers.WatcherDef{
			Name:              "remediate",
			Type:              "exec",
			StateMatch:        []string{"CRITICAL"},
			SuccessTransition: "UNKNOWN",
			Interval:          c.check.RemediateInterval.String(),
			Properties: map[string]interface{}{
				"command": c.check.RemediateCommand,
			},
		})
	}

	cj, err := json.Marshal(&check)
	if err != nil {
		return err
	}

	err = os.MkdirAll(c.machineDir, 0755)
	if err != nil {
		return err
	}

	tf, err := ioutil.TempFile(c.machineDir, "")
	if err != nil {
		return err
	}
	tf.Close()

	err = ioutil.WriteFile(tf.Name(), cj, 0644)
	if err != nil {
		return err
	}

	err = os.Rename(tf.Name(), filepath.Join(c.machineDir, "machine.yaml"))
	if err != nil {
		return err
	}

	return nil
}

func (c *Manager) Set(check *Check) error {
	cj, err := json.Marshal(check)
	if err != nil {
		return err
	}

	_, err = c.nc.Request(c.subj, cj, 5*time.Second)
	if err != nil {
		return err
	}

	return nil
}

func (c *Manager) Check() *Check {
	return c.check
}
