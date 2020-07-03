package tags

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/scout/stream"
	"github.com/choria-io/go-choria/scout/updatenotifier"
)

type CheckList []string

type Manager struct {
	checks *CheckList
	tag    string
	nc     *nats.Conn
	mgr    *stream.Mgr
	subj   string

	*logrus.Entry
	updatenotifier.Notifier
	sync.Mutex
}

type Framework interface {
	NATSConn() *nats.Conn
	Logger(string) *logrus.Entry
}

func NewManager(tag string, fw Framework) (*Manager, error) {
	if tag == "" {
		return nil, fmt.Errorf("invalid tag %q", tag)
	}

	c := &Manager{
		checks: &CheckList{},
		tag:    tag,
		nc:     fw.NATSConn(),
		subj:   "scout.tags." + tag,
		Entry:  fw.Logger("tag").WithField("tag", tag),
	}

	mgr, err := stream.New("SCOUT_TAGS", c.subj, fw)
	if err != nil {
		return nil, err
	}

	c.mgr = mgr

	return c, nil
}

func (c *Manager) Set(checks ...string) error {
	cj, err := json.Marshal(checks)
	if err != nil {
		return err
	}

	_, err = c.nc.Request(c.subj, cj, 5*time.Second)
	if err != nil {
		return err
	}

	return nil
}

func (c *Manager) Start(ctx context.Context, wg *sync.WaitGroup) error {
	c.Infof("Listening for tag updates")

	wg.Add(1)
	go c.Notifier.Update(ctx, wg)

	return c.mgr.Manage(c)
}

// Instance implements stream.updatable
func (c *Manager) Instance() interface{} {
	return &CheckList{}
}

// Update implements stream.updatable
func (c *Manager) Update(u interface{}) {
	c.Debugf("received tag update %v", u)

	update, ok := u.(*CheckList)
	if !ok {
		return
	}

	c.Lock()
	c.checks = update
	c.Unlock()

	c.Notify()
}

func (c *Manager) Checks() CheckList {
	c.Lock()
	defer c.Unlock()

	return *c.checks
}
