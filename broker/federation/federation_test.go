package federation

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/inter"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
)

var c *choria.Framework

func init() {
	os.Setenv("MCOLLECTIVE_CERTNAME", "rip.mcollective")
	cfg, _ := config.NewConfig("testdata/federation.cfg")
	cfg.OverrideCertname = "rip.mcollective"
	c, _ = choria.NewWithConfig(cfg)
}

func TestFederation(t *testing.T) {
	log.SetOutput(io.Discard)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker/Federation")
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
	SubChannels map[string]chan inter.ConnectorMessage
	mu          *sync.Mutex
}

func (s *stubConnection) PublishToQueueSub(name string, msg inter.ConnectorMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.SubChannels[name]
	if !ok {
		s.SubChannels[name] = make(chan inter.ConnectorMessage, 1000)
		c = s.SubChannels[name]
	}

	c <- msg
}

func (s *stubConnection) AgentBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.agent.%s", collective, agent)
}

func (s *stubConnection) ServiceBroadcastTarget(collective string, agent string) string {
	return fmt.Sprintf("%s.broadcast.service.%s", collective, agent)
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

func (s *stubConnection) IsConnected() bool { return true }

func (s *stubConnection) Unsubscribe(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Subs, name)
	delete(s.SubChannels, name)

	return nil
}

func (s *stubConnection) ChanQueueSubscribe(name string, subject string, group string, capacity int) (chan inter.ConnectorMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Subs[name] = [3]string{name, subject, group}

	_, ok := s.SubChannels[name]
	if !ok {
		s.SubChannels[name] = make(chan inter.ConnectorMessage, 1000)
	}

	return s.SubChannels[name], nil
}

func (s *stubConnection) QueueSubscribe(ctx context.Context, name string, subject string, group string, output chan inter.ConnectorMessage) error {
	return nil
}

func (s *stubConnection) PublishRaw(target string, data []byte) error {
	s.Outq <- [2]string{target, string(data)}

	return nil
}

func (s *stubConnection) Publish(msg inter.Message) error {
	return nil
}

func (s *stubConnection) Connect(ctx context.Context) error {
	return nil
}

func (s *stubConnection) Close() {}

func (s *stubConnection) ReplyTarget(msg inter.Message) (string, error) {
	return "stubreplytarget", nil
}

func (s *stubConnection) Nats() *nats.Conn {
	return &nats.Conn{}
}

func (s *stubConnection) PublishRawMsg(msg *nats.Msg) error { return fmt.Errorf("not implemented") }
func (s *stubConnection) RequestRawMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *stubConnectionManager) NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *log.Entry) (conn inter.Connector, err error) {
	if s.connection != nil {
		return s.connection, nil
	}

	conn = &stubConnection{
		Outq:        make(chan [2]string, 64),
		SubChannels: make(map[string]chan inter.ConnectorMessage),
		Subs:        make(map[string][3]string),
		mu:          &sync.Mutex{},
	}

	s.connection = conn.(*stubConnection)

	return
}

func (s *stubConnectionManager) Init() *stubConnectionManager {
	s.connection = &stubConnection{
		Outq:        make(chan [2]string, 64),
		SubChannels: make(map[string]chan inter.ConnectorMessage),
		Subs:        make(map[string][3]string),
		mu:          &sync.Mutex{},
	}

	return s
}

var _ = Describe("Federation Broker", func() {
	It("Should initialize correctly", func() {
		log.SetOutput(io.Discard)

		c, err := choria.New("testdata/federation.cfg")
		Expect(err).ToNot(HaveOccurred())

		_, err = NewFederationBroker("test_cluster", c)
		Expect(err).ToNot(HaveOccurred())
	})
})
