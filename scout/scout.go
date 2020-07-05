package scout

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/srvcache"
)

type Scout struct {
	cfg    *config.Config
	choria Choria
	conn   Connector
	*logrus.Entry
	entity *Entity

	started bool

	sync.Mutex
}

type Connector interface {
	Nats() *nats.Conn
}

type Choria interface {
	Configuration() *config.Config
	Logger(string) *logrus.Entry
	NewConnector(ctx context.Context, servers func() (srvcache.Servers, error), name string, logger *logrus.Entry) (conn choria.Connector, err error)
	MiddlewareServers() (servers srvcache.Servers, err error)
	Certname() string
}

func New(fw Choria) (*Scout, error) {
	s := &Scout{
		cfg:    fw.Configuration(),
		choria: fw,
		Entry:  fw.Logger("scout"),
	}

	return s, nil
}

func (s *Scout) Start(ctx context.Context, wg *sync.WaitGroup) error {
	s.Lock()
	defer s.Unlock()

	if s.started {
		return fmt.Errorf("already started")
	}

	conn, err := s.choria.NewConnector(ctx, s.choria.MiddlewareServers, s.choria.Certname(), s.Entry)
	if err != nil {
		return err
	}
	s.conn = conn

	entity, err := NewEntity(ctx, wg, s)
	if err != nil {
		return err
	}

	s.entity = entity
	s.started = true

	return nil
}

func (s *Scout) MachineSourceDir() string {
	return s.cfg.Choria.MachineSourceDir
}

func (s *Scout) Logger(component string) *logrus.Entry {
	return s.choria.Logger(component)
}

func (s *Scout) Tags() ([]string, error) {
	if s.cfg.Choria.ScoutTags == "" {
		return nil, fmt.Errorf("tags file is not set")
	}

	tb, err := ioutil.ReadFile(s.cfg.Choria.ScoutTags)
	if err != nil {
		return nil, err
	}

	var tags []string
	err = json.Unmarshal(tb, &tags)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (s *Scout) OverridesFile() string {
	return s.cfg.Choria.ScoutOverrides
}

func (s *Scout) NATSConn() *nats.Conn {
	return s.conn.Nats()
}

func (s *Scout) Identity() string {
	return s.cfg.Identity
}