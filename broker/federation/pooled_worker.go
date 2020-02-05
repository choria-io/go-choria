package federation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
	"github.com/choria-io/go-protocol/protocol"
	log "github.com/sirupsen/logrus"
)

type chainable interface {
	Name() string
	From(input chainable) error
	To(output chainable) error
	Input() chan chainmessage
	Output() chan chainmessage
}

type runable interface {
	Init(workers int, broker *FederationBroker) error
	Run(ctx context.Context) error
	Ready() bool
}

type chainmessage struct {
	Targets   []string
	RequestID string
	Message   protocol.TransportMessage
	Seen      []string
}

type pooledWorker struct {
	name        string
	in          chan chainmessage
	out         chan chainmessage
	initialized bool
	broker      *FederationBroker
	mode        int
	capacity    int
	workers     int
	mu          sync.Mutex
	log         *log.Entry
	wg          *sync.WaitGroup

	choria     *choria.Framework
	connection choria.ConnectionManager
	servers    func() (srvcache.Servers, error)

	worker func(ctx context.Context, w *pooledWorker, instance int, logger *log.Entry)
}

func PooledWorkerFactory(name string, workers int, mode int, capacity int, broker *FederationBroker, logger *log.Entry, worker func(context.Context, *pooledWorker, int, *log.Entry)) (*pooledWorker, error) {
	w := &pooledWorker{
		name:     name,
		mode:     mode,
		log:      logger,
		worker:   worker,
		capacity: capacity,
		wg:       &sync.WaitGroup{},
	}

	err := w.Init(workers, broker)

	return w, err
}

func (w *pooledWorker) Run(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.Ready() {
		err := fmt.Errorf("could not run %s as Init() has not been called or failed", w.Name())
		w.log.Warn(err)
		return err
	}

	var err error

	if w.mode != Unconnected {
		switch w.mode {
		case Federation:
			w.servers = w.choria.FederationMiddlewareServers
		case Collective:
			w.servers = w.choria.MiddlewareServers
		default:
			err := errors.New("do not know which middleware to connect to, Mode should be one of Federation or Collective")
			w.log.Error(err)
			return err
		}

		if err != nil {
			err = fmt.Errorf("could not determine middleware servers: %s", err)
			w.log.Warn(err)
			return err
		}

		srv, err := w.servers()
		if err != nil {
			err = fmt.Errorf("resolving initial middleware server list failed: %s", err)
			w.log.Error(err)
			return err
		}

		if srv.Count() == 0 {
			err = fmt.Errorf("no middleware servers were configured for %s, cannot continue", w.name)
			w.log.Error(err)
			return err
		}
	}

	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)

		go w.worker(ctx, w, i, w.log.WithFields(log.Fields{"worker_instance": i}))
	}

	w.wg.Wait()

	return nil
}

func (w *pooledWorker) Init(workers int, broker *FederationBroker) (err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.workers = workers
	w.choria = broker.choria
	w.broker = broker

	if w.mode != Unconnected {
		w.connection = broker.choria
	}

	if w.log == nil {
		w.log = broker.logger.WithFields(log.Fields{"worker": w.name})
	}

	if w.capacity == 0 {
		w.capacity = 100
	}

	if w.workers == 0 {
		w.workers = 2
	}

	w.in = make(chan chainmessage, w.capacity)
	w.out = make(chan chainmessage, w.capacity)

	w.initialized = true

	return nil
}

func (w *pooledWorker) Ready() bool {
	return w.initialized
}

func (w *pooledWorker) Name() string {
	return w.name
}

func (w *pooledWorker) From(input chainable) error {
	if input.Output() == nil {
		return fmt.Errorf("Input %s does not have a output chain", input.Name())
	}

	w.log.Debugf("Connecting input of %s to output of %s with capacity %d", w.Name(), input.Name(), cap(input.Output()))

	w.in = input.Output()

	return nil
}

func (w *pooledWorker) To(output chainable) error {
	if output.Input() == nil {
		return fmt.Errorf("Output %s does not have a input chain", output.Name())
	}

	w.log.Debugf("Connecting output of %s to input of %s with capacity %d", w.Name(), output.Name(), cap(output.Input()))

	w.out = output.Input()

	return nil
}

func (w *pooledWorker) Input() chan chainmessage {
	return w.in
}

func (w *pooledWorker) Output() chan chainmessage {
	return w.out
}
