package stream

//
// import (
// 	"encoding/json"
// 	"fmt"
// 	"sync"
// 	"time"
//
// 	"github.com/nats-io/jsm.go"
// 	"github.com/nats-io/nats.go"
// 	"github.com/sirupsen/logrus"
//
// 	"github.com/choria-io/go-choria/choria"
// )
//
// type updatable interface {
// 	Instance() interface{}
// 	Update(interface{})
// }
//
// type Framework interface {
// 	NATSConn() *nats.Conn
// 	Logger(string) *logrus.Entry
// }
//
// type Mgr struct {
// 	nc     *nats.Conn
// 	stream string
// 	subj   string
// 	log    *logrus.Entry
// 	mgr    *jsm.Manager
// 	eph    *Ephemeral
// 	q      chan *choria.ConnectorMessage
//
// 	sync.Mutex
// }
//
// // New creates a new stream based data manager
// func New(stream string, subs string, fw Framework) (*Mgr, error) {
// 	mgr, err := jsm.New(fw.NATSConn())
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	m := &Mgr{
// 		nc:     fw.NATSConn(),
// 		stream: stream,
// 		subj:   subs,
// 		mgr:    mgr,
// 		q:      make(chan *choria.ConnectorMessage, 1000),
// 		log:    fw.Logger("stream").WithFields(logrus.Fields{"stream": stream, "subjects": subs}),
// 	}
//
// 	return m, nil
// }
//
// func (m *Mgr) Close() {
// 	m.Lock()
// 	eph := m.eph
// 	close(m.q)
// 	m.Unlock()
//
// 	if eph != nil {
// 		eph.Close()
// 	}
// }
//
// func (m *Mgr) Manage(d updatable) error {
// 	str, err := m.mgr.LoadStream(m.stream)
// 	if err != nil {
// 		return fmt.Errorf("could not load stream %s: %s", m.stream, err)
// 	}
//
// 	go func() {
// 		for cm := range m.q {
// 			m.Lock()
// 			defer m.Unlock()
//
// 			msg, ok := cm.Msg.(*nats.Msg)
// 			if !ok {
// 				continue
// 			}
//
// 			if m.eph != nil {
// 				m.eph.SetResumeSequence(msg)
// 			}
//
// 			t := d.Instance()
// 			err = json.Unmarshal(msg.Data, t)
// 			if err != nil {
// 				m.log.Errorf("failed to handle incoming update: %s", err)
// 				return
// 			}
//
// 			d.Update(t)
//
// 			err = msg.Respond(nil)
// 			if err != nil {
// 				m.log.Errorf("failed to acknowledge update: %s", err)
// 			}
// 		}
// 	}()
//
// 	m.eph, err = NewEphemeral(m.nc, str, time.Hour, m.q, m.log, jsm.FilterStreamBySubject(m.subj), jsm.StartWithLastReceived())
// 	if err != nil {
// 		return err
// 	}
//
// 	return nil
// }
