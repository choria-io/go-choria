package federation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/choria"
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
	servers    func() ([]choria.Server, error)

	worker func(ctx context.Context, self *pooledWorker, instance int, logger *log.Entry)
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

func (self *pooledWorker) Run(ctx context.Context) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.Ready() {
		err := fmt.Errorf("Could not run %s as Init() has not been called or failed", self.Name())
		self.log.Warn(err)
		return err
	}

	var err error

	if self.mode != Unconnected {
		// look up here so it hits the name servers once only
		switch self.mode {
		case Federation:
			self.servers = func() ([]choria.Server, error) {
				return self.choria.FederationMiddlewareServers()
			}
		case Collective:
			self.servers = func() ([]choria.Server, error) {
				return self.choria.MiddlewareServers()
			}
		default:
			err := errors.New("Do not know which middleware to connect to, Mode should be one of Federation or Collective")
			self.log.Warn(err)
			return err
		}

		if err != nil {
			err = fmt.Errorf("Could not determine middleware servers: %s", err)
			self.log.Warn(err)
			return err
		}
	}

	for i := 0; i < self.workers; i++ {
		self.wg.Add(1)

		go self.worker(ctx, self, i, self.log.WithFields(log.Fields{"worker_instance": i}))
	}

	self.wg.Wait()

	return nil
}

func (self *pooledWorker) Init(workers int, broker *FederationBroker) (err error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.workers = workers
	self.choria = broker.choria
	self.broker = broker

	if self.mode != Unconnected {
		self.connection = broker.choria
	}

	if self.log == nil {
		self.log = broker.logger.WithFields(log.Fields{"worker": self.name})
	}

	if self.capacity == 0 {
		self.capacity = 100
	}

	if self.workers == 0 {
		self.workers = 2
	}

	self.in = make(chan chainmessage, self.capacity)
	self.out = make(chan chainmessage, self.capacity)

	self.initialized = true

	return nil
}

func (self *pooledWorker) Ready() bool {
	return self.initialized
}

func (self *pooledWorker) Name() string {
	return self.name
}

func (self *pooledWorker) From(input chainable) error {
	if input.Output() == nil {
		return fmt.Errorf("Input %s does not have a output chain", input.Name())
	}

	self.log.Debugf("Connecting input of %s to output of %s with capacity %d", self.Name(), input.Name(), cap(input.Output()))

	self.in = input.Output()

	return nil
}

func (self *pooledWorker) To(output chainable) error {
	if output.Input() == nil {
		return fmt.Errorf("Output %s does not have a input chain", output.Name())
	}

	self.log.Debugf("Connecting output of %s to input of %s with capacity %d", self.Name(), output.Name(), cap(output.Input()))

	self.out = output.Input()

	return nil
}

func (self *pooledWorker) Input() chan chainmessage {
	return self.in
}

func (self *pooledWorker) Output() chan chainmessage {
	return self.out
}
