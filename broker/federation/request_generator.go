package federation

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/mcollective"
	"github.com/choria-io/go-choria/protocol"
	log "github.com/sirupsen/logrus"
)

// RequestGenerator is a Receiver that generates many Request
type RequestGenerator struct {
	chainbase
	choria *mcollective.Choria

	mu sync.Mutex
}

func (self *RequestGenerator) work(i int, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		req, err := self.choria.NewRequest(protocol.RequestV1, "discovery", "generator", "choria=generator", 60, self.choria.NewRequestID(), "mcollective")
		if err != nil {
			log.Warnf("Could not generate a new request: %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		req.SetMessage(`{"hello":"world"}`)

		sr, err := self.choria.NewSecureRequest(req)
		if err != nil {
			log.Warnf("Could not generate a new secure request: %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		t, err := self.choria.NewTransportForSecureRequest(sr)
		if err != nil {
			log.Warnf("Could not generate a new transport message: %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		t.SetFederationRequestID(req.RequestID())
		t.SetFederationTargets([]string{"foo.request"})
		t.SetReplyTo("foo.reply")

		cm := chainmessage{
			Targets:   []string{},
			RequestID: req.RequestID(),
			Message:   t,
		}

		self.out <- cm

		log.Infof("%d Generated request %s (%d)", i, req.RequestID(), len(self.out))
	}
}

func (self *RequestGenerator) Run() error {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.Ready() {
		return errors.New("Could not run RequestGenerator as Init() has not been called or failed")
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go self.work(i, &wg)
	}

	wg.Wait()

	return nil
}

func (self *RequestGenerator) Init(cluster string, instance string) (err error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	fmt.Println(mcollective.UserConfig())

	self.choria, err = mcollective.New(mcollective.UserConfig())
	if err != nil {
		err = fmt.Errorf("Could not initialize RequestGenerator: %s", err.Error())
		return
	}

	self.out = make(chan chainmessage, 1000)
	self.name = fmt.Sprintf("%s:%s Request Generator", cluster, instance)
	self.initialized = true

	return
}
