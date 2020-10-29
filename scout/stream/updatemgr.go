package stream

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type updatable interface {
	Instance() interface{}
	Update(interface{})
}

type Framework interface {
	NATSConn() *nats.Conn
	Logger(string) *logrus.Entry
}

type Mgr struct {
	nc     *nats.Conn
	stream string
	subj   string
	log    *logrus.Entry
	mgr    *jsm.Manager

	sync.Mutex
}

// New creates a new stream based data manager
func New(stream string, subs string, fw Framework) (*Mgr, error) {
	mgr, err := jsm.New(fw.NATSConn())
	if err != nil {
		return nil, err
	}

	return &Mgr{
		nc:     fw.NATSConn(),
		stream: stream,
		subj:   subs,
		mgr:    mgr,
		log:    fw.Logger("stream").WithFields(logrus.Fields{"stream": stream, "subjects": subs}),
	}, nil
}

func (m *Mgr) Manage(d updatable) error {
	str, err := m.mgr.LoadStream(m.stream)
	if err != nil {
		return fmt.Errorf("could not load stream %s: %s", m.stream, err)
	}

	sub, err := m.nc.Subscribe(nats.NewInbox(), func(msg *nats.Msg) {
		m.Lock()
		defer m.Unlock()

		t := d.Instance()
		err = json.Unmarshal(msg.Data, t)
		if err != nil {
			m.log.Errorf("failed to handle incoming update: %s", err)
			return
		}

		d.Update(t)

		err = msg.Respond(nil)
		if err != nil {
			m.log.Errorf("failed to acknowledge update: %s", err)
		}
	})
	if err != nil {
		return err
	}

	_, err = str.NewConsumer(jsm.FilterStreamBySubject(m.subj), jsm.StartWithLastReceived(), jsm.DeliverySubject(sub.Subject))
	if err != nil {
		return err
	}

	return nil
}
