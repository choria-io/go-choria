package aagent

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/aagent/machine"
	notifier "github.com/choria-io/go-choria/aagent/notifiers/choria"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type AAgent struct {
	fw       ChoriaProvider
	logger   *logrus.Entry
	machines []*managedMachine
	manager  *watchers.Manager
	notifier *notifier.Notifier

	source string

	sync.Mutex
}

type managedMachine struct {
	path    string
	loaded  time.Time
	machine *machine.Machine
}

// ChoriaProvider provides access to the choria framework
type ChoriaProvider interface {
	PublishRaw(string, []byte) error
	Logger(string) *logrus.Entry
	Identity() string
}

// New creates a new instance of the choria autonomous agent host
func New(dir string, fw ChoriaProvider) (aa *AAgent, err error) {
	notifier, err := notifier.New(fw)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create notifier")
	}

	return &AAgent{
		fw:       fw,
		logger:   fw.Logger("aagent"),
		source:   dir,
		machines: []*managedMachine{},
		manager:  watchers.New(),
		notifier: notifier,
	}, nil
}

func (a *AAgent) InitialLoadMachines(ctx context.Context, wg *sync.WaitGroup) error {
	files, err := ioutil.ReadDir(a.source)
	if err != nil {
		return errors.Wrapf(err, "could not read %s", a.source)
	}

	for _, file := range files {
		path := filepath.Join(a.source, file.Name())

		if !file.IsDir() || strings.HasPrefix(path, ".") {
			continue
		}

		a.logger.Infof("Attempting to load Choria Machine from %s", path)

		err = a.loadMachine(ctx, wg, path)
		if err != nil {
			a.logger.Errorf("Could not load machine from %s: %s", path, err)
		}
	}

	return nil
}

func (a *AAgent) loadMachine(ctx context.Context, wg *sync.WaitGroup, path string) (err error) {
	machine, err := machine.FromDir(path, a.manager)
	if err != nil {
		return err
	}

	machine.SetIdentity(a.fw.Identity())
	machine.RegisterNotifier(a.notifier)

	managed := &managedMachine{
		loaded:  time.Now(),
		path:    path,
		machine: machine,
	}

	a.Lock()
	a.machines = append(a.machines, managed)
	a.Unlock()

	machine.Start(ctx, wg)

	return nil
}
