package data

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/internal/util"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/data/ddl"
	"github.com/choria-io/go-choria/server/agents"
)

type Query interface{}
type OutputItem interface{}

type Creator struct {
	F    func(Framework) (Plugin, error)
	Name string
}

type Plugin interface {
	Run(context.Context, Query, agents.ServerInfoSource) (map[string]OutputItem, error)
	DLL() (*ddl.DDL, error)
}

type Framework interface {
	Configuration() *config.Config
	Logger(string) *logrus.Entry
}

type Manager struct {
	plugins map[string]*Creator
	ddls    map[string]*ddl.DDL
	fw      Framework
	ctx     context.Context

	log *logrus.Entry
}

var (
	plugins map[string]*Creator
	mu      sync.Mutex
)

func NewManager(ctx context.Context, fw Framework) (*Manager, error) {
	m := &Manager{
		fw:      fw,
		plugins: make(map[string]*Creator),
		ddls:    make(map[string]*ddl.DDL),
		log:     fw.Logger("data_manager"),
		ctx:     ctx,
	}

	mu.Lock()
	for k, p := range plugins {
		pi, err := p.F(fw)
		if err != nil {
			m.log.Warnf("Could not create instance of data plugin %s", k)
			continue
		}

		ddl, err := pi.DLL()
		if err != nil {
			m.log.Warnf("Could not load DDL for data plugin %s", k)
			continue
		}
		m.plugins[k] = p
		m.ddls[k] = ddl
		m.log.Infof("Activated Data provider %s", k)
	}
	mu.Unlock()

	return m, nil
}

func RegisterPlugin(name string, plugin *Creator) error {
	mu.Lock()
	defer mu.Unlock()

	if plugins == nil {
		plugins = make(map[string]*Creator)
	}

	_, ok := plugins[plugin.Name]
	if ok {
		return fmt.Errorf("data plugin %s is already registered", plugin.Name)
	}

	plugins[plugin.Name] = plugin
	util.BuildInfo().RegisterDataProvider(name)

	return nil
}

// DDLs is a list of DDLs for all known data plugins
func (m *Manager) DDLs() []*ddl.DDL {
	res := []*ddl.DDL{}
	for _, v := range m.ddls {
		res = append(res, v)
	}

	return res
}

func (m *Manager) FuncMap(si agents.ServerInfoSource) (ddl.FuncMap, error) {
	funcs := make(ddl.FuncMap)

	for k, v := range m.ddls {
		entry := ddl.FuncMapEntry{DDL: v, Name: v.Metadata.Name}
		if v.Query == nil {
			entry.F = m.arityZeroRunner(m.ctx, si, k, v, m.plugins[k])
		} else {
			entry.F = m.arityOneRunner(m.ctx, si, k, v, m.plugins[k])
		}
		funcs[k] = entry
	}

	return funcs, nil
}

func (m *Manager) arityZeroRunner(ctx context.Context, si agents.ServerInfoSource, name string, ddl *ddl.DDL, plugin *Creator) func() map[string]OutputItem {
	return func() map[string]OutputItem {
		f := m.arityOneRunner(ctx, si, name, ddl, plugin)
		return f("")
	}
}

func (m *Manager) arityOneRunner(ctx context.Context, si agents.ServerInfoSource, name string, ddl *ddl.DDL, plugin *Creator) func(q string) map[string]OutputItem {
	return func(q string) map[string]OutputItem {
		ctx, cancel := context.WithTimeout(ctx, ddl.Timeout())
		defer cancel()

		i, err := plugin.F(m.fw)
		if err != nil {
			m.log.Errorf("Could not create an instance of data provider %s: %s", name, err)
			return nil
		}

		result, err := i.Run(ctx, q, si)
		if err != nil {
			m.log.Errorf("Could not run data provider %s: %s", name, err)
			return nil
		}

		return result
	}
}

func (m *Manager) Execute(ctx context.Context, plugin string, query string, srv agents.ServerInfoSource) (map[string]OutputItem, error) {
	dpc, ok := m.plugins[plugin]
	if !ok {
		return nil, fmt.Errorf("unknown data plugin %s", plugin)
	}

	dp, err := dpc.F(m.fw)
	if err != nil {
		return nil, fmt.Errorf("could not create instance of data plugin %s: %s", plugin, err)
	}

	pddl, ok := m.ddls[plugin]
	if !ok {
		return nil, fmt.Errorf("could not find DDL for plugin %s", plugin)
	}

	var (
		q interface{}
	)

	if pddl.Query != nil {
		q, err = pddl.Query.ConvertStringValue(query)
		if err != nil {
			return nil, err
		}

		w, err := pddl.Query.ValidateValue(q)
		if err != nil {
			return nil, err
		}

		if len(w) > 0 {
			return nil, fmt.Errorf("invalid query: %s", strings.Join(w, ", "))
		}
	}

	timeout, cancel := context.WithTimeout(ctx, pddl.Timeout())
	defer cancel()

	return dp.Run(timeout, q, srv)
}
