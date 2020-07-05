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

	sync.Mutex
}

// New creates a new stream based data manager
func New(stream string, subs string, fw Framework) (*Mgr, error) {
	return &Mgr{
		nc:     fw.NATSConn(),
		stream: stream,
		subj:   subs,
		log:    fw.Logger("stream").WithFields(logrus.Fields{"stream": stream, "subjects": subs}),
	}, nil
}

func (m *Mgr) Manage(d updatable) error {
	str, err := jsm.LoadStream(m.stream, jsm.WithConnection(m.nc))
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
		msg.Respond(nil)
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
