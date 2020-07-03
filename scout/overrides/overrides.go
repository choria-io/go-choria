package overrides

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/logger"
	"github.com/choria-io/go-choria/scout/stream"
	"github.com/choria-io/go-choria/scout/updatenotifier"
)

type Override map[string]interface{}

type Manager struct {
	data *Override
	name string
	mgr  *stream.Mgr
	subj string
	nc   *nats.Conn

	logger.Logrus
	updatenotifier.Notifier
	sync.Mutex
}

type Framework interface {
	Identity() string
	NATSConn() *nats.Conn
	Logger(string) *logrus.Entry
}

func NewManager(fw Framework) (*Manager, error) {
	o := &Manager{
		name:   fw.Identity(),
		subj:   "scout.overrides." + fw.Identity(),
		nc:     fw.NATSConn(),
		Logrus: fw.Logger("overrides").WithField("identity", fw.Identity()),
	}

	mgr, err := stream.New("SCOUT_OVERRIDES", o.subj, fw)
	if err != nil {
		return nil, err
	}

	o.mgr = mgr

	return o, nil
}

func (o *Manager) Start(ctx context.Context, wg *sync.WaitGroup) error {
	o.Infof("Listening for override updates")

	wg.Add(1)
	go o.Notifier.Update(ctx, wg)

	return o.mgr.Manage(o)

}

// Instance implements stream.updatable
func (o *Manager) Instance() interface{} {
	return &Override{}
}

// Update implements stream.updatable
func (o *Manager) Update(u interface{}) {
	o.Debugf("received update notification %v", u)

	update, ok := u.(*Override)
	if !ok {
		return
	}

	o.Lock()
	o.data = update
	o.Unlock()

	o.Notify()
}

func (o *Manager) Set(overrides Override) error {
	cj, err := json.Marshal(overrides)
	if err != nil {
		return err
	}

	_, err = o.nc.Request(o.subj, cj, 5*time.Second)
	if err != nil {
		return err
	}

	return nil
}

func (o *Manager) JSON() ([]byte, error) {
	o.Lock()
	defer o.Unlock()

	return json.Marshal(o.data)
}
