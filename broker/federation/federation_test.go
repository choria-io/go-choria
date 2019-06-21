package federation

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var c *choria.Framework

func init() {
	c, _ = choria.New("testdata/federation.cfg")
}

func newDiscardLogger() (*log.Entry, *bufio.Writer, *bytes.Buffer) {
	var logbuf bytes.Buffer

	logger := log.New().WithFields(log.Fields{"test": "true"})
	logger.Logger.Level = log.DebugLevel
	logtxt := bufio.NewWriter(&logbuf)
	logger.Logger.Out = logtxt

	return logger, logtxt, &logbuf
}

func waitForLogLines(w *bufio.Writer, b *bytes.Buffer) {
	for {
		w.Flush()
		if b.Len() > 0 {
			return
		}
	}

}

type stubConnectionManager struct {
	connection *stubConnection
}

type stubConnection struct {
	Outq        chan [2]string
	Subs        map[string][3]string
	SubChannels map[string]chan *choria.ConnectorMessage
	name        string
	mu          *sync.Mutex
}

func (s *stubConnection) PublishToQueueSub(name string, msg *choria.ConnectorMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.SubChannels[name]
	if !ok {
		s.SubChannels[name] = make(chan *choria.ConnectorMessage, 1000)
		c = s.SubChannels[name]
	}

	c <- msg
}

func (s *stubConnection) AgentBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.agent.%s", collective, agent)
}

func (s *stubConnection) NodeDirectedTarget(collective string, identity string) string {
	return fmt.Sprintf("%s.node.%s", collective, identity)
}

func (s *stubConnection) ConnectedServer() string {
	return "nats://stub:4222"
}

func (s *stubConnection) ConnectionOptions() nats.Options {
	return nats.Options{}
}

func (s *stubConnection) ConnectionStats() nats.Statistics {
	return nats.Statistics{}
}

func (s *stubConnection) Unsubscribe(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.Subs[name]; ok {
		delete(s.Subs, name)
	}

	if _, ok := s.SubChannels[name]; ok {
		delete(s.SubChannels, name)
	}

	return nil
}

func (s *stubConnection) ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan *choria.ConnectorMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Subs[name] = [3]string{name, subject, group}

	_, ok := s.SubChannels[name]
	if !ok {
		s.SubChannels[name] = make(chan *choria.ConnectorMessage, 1000)
	}

	return s.SubChannels[name], nil
}

func (s *stubConnection) QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan *choria.ConnectorMessage) error {
	return nil
}

func (s *stubConnection) PublishRaw(target string, data []byte) error {
	s.Outq <- [2]string{target, string(data)}

	return nil
}

func (s *stubConnection) Publish(msg *choria.Message) error {
	return nil
}

func (s *stubConnection) Connect(ctx context.Context) error {
	return nil
}

func (s *stubConnection) Close() {
	return
}

func (s *stubConnection) ReplyTarget(msg *choria.Message) (string, error) {
	return "stubreplytarget", nil
}

func (s *stubConnection) Nats() *nats.Conn {
	return &nats.Conn{}
}

func (s *stubConnectionManager) NewConnector(ctx context.Context, servers func() ([]srvcache.Server, error), name string, logger *log.Entry) (conn choria.Connector, err error) {
	if s.connection != nil {
		return s.connection, nil
	}

	conn = &stubConnection{
		Outq:        make(chan [2]string, 64),
		SubChannels: make(map[string]chan *choria.ConnectorMessage),
		Subs:        make(map[string][3]string),
		mu:          &sync.Mutex{},
	}

	s.connection = conn.(*stubConnection)

	return
}

func (s *stubConnectionManager) Init() *stubConnectionManager {
	s.connection = &stubConnection{
		Outq:        make(chan [2]string, 64),
		SubChannels: make(map[string]chan *choria.ConnectorMessage),
		Subs:        make(map[string][3]string),
		mu:          &sync.Mutex{},
	}

	return s
}
func TestFederation(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")

	RegisterFailHandler(Fail)
	RunSpecs(t, "Federation")
}

var _ = Describe("Federation Broker", func() {
	It("Should initialize correctly", func() {
		log.SetOutput(ioutil.Discard)

		c, err := choria.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		_, err = NewFederationBroker("test_cluster", c)
		Expect(err).ToNot(HaveOccurred())
	})
})
