package federation

import (
	"fmt"

	"github.com/choria-io/go-choria/protocol"
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
	Init(cluster string, instance string) error
	Run() error
	Ready() bool
}

type chainmessage struct {
	Targets   []string
	RequestID string
	Message   protocol.TransportMessage
}

type chainbase struct {
	name        string
	in          chan chainmessage
	out         chan chainmessage
	initialized bool
}

func (self *chainbase) Ready() bool {
	return self.initialized
}

func (self *chainbase) Name() string {
	return self.name
}

func (self *chainbase) From(input chainable) error {
	if input.Output() == nil {
		return fmt.Errorf("Input %s does not have a output chain", input.Name())
	}

	log.Infof("Connecting %s -> %s with capacity %d", input.Name(), self.Name(), cap(input.Output()))
	self.in = input.Output()

	return nil
}

func (self *chainbase) To(output chainable) error {
	if output.Input() == nil {
		return fmt.Errorf("Output %s does not have a input chain", output.Name())
	}

	log.Infof("Connecting %s -> %s with capacity %d", self.Name(), output.Name(), cap(output.Input()))
	self.out = output.Input()

	return nil
}

func (self *chainbase) Input() chan chainmessage {
	return self.in
}

func (self *chainbase) Output() chan chainmessage {
	return self.out
}
