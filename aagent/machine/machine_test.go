// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package machine

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/plugin"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aagent/Machine")
}

var _ = Describe("Aagent/Machine", func() {
	var (
		mockctl *gomock.Controller
		service *MockNotificationService
		manager *MockWatcherManager
		machine *Machine
		log     *logrus.Entry
		err     error
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.SetOutput(GinkgoWriter)
		log = logrus.NewEntry(logger)

		mockctl = gomock.NewController(GinkgoT())
		service = NewMockNotificationService(mockctl)
		manager = NewMockWatcherManager(mockctl)
		machine = &Machine{
			notifiers:   []NotificationService{},
			manager:     manager,
			MachineName: "ginkgo",
		}
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("FromYAML", func() {
		It("Should configure the manager", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{})).Return(fmt.Errorf("set machine error"))
			machine, err = FromYAML("testdata/empty.yaml", manager)
			Expect(err).To(MatchError("could not register with manager: set machine error"))
		})

		It("Should setup the machine", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{}))
			machine, err = FromYAML("testdata/empty.yaml", manager)
			Expect(err).To(MatchError("validation failed: a machine name is required"))
		})

		It("Should load good machines", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{}))
			machine, err = FromYAML("testdata/machine.yaml", manager)
			Expect(err).ToNot(HaveOccurred())
			Expect(machine.Name()).To(Equal("TestMachine"))
		})
	})

	Describe("FromPlugin", func() {
		var (
			mplug *plugin.MachinePlugin
		)
		BeforeEach(func() {
			myaml, err := os.ReadFile("testdata/machine.yaml")
			Expect(err).ToNot(HaveOccurred())

			Expect(yaml.Unmarshal(myaml, machine)).ToNot(HaveOccurred())
			mplug = plugin.NewMachinePlugin("TestMachine", machine)
		})

		It("Should configure the manager and handle errors", func() {
			manager.EXPECT().SetMachine(gomock.Any()).DoAndReturn(func(m *Machine) error {
				Expect(m.MachineName).To(Equal("TestMachine"))
				Expect(m.manager).To(Equal(manager))

				return fmt.Errorf("set machine error")
			})

			machine, err = FromPlugin(mplug, manager, log)
			Expect(err).To(MatchError("could not register with manager: set machine error"))
		})

		It("Should setup the machine", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(machine))
			machine.MachineName = ""
			machine, err = FromPlugin(mplug, manager, log)
			Expect(err).To(MatchError("validation failed: a machine name is required"))
		})

		It("Should load good machines", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(machine))
			machine, err = FromPlugin(mplug, manager, log)
			Expect(err).ToNot(HaveOccurred())
			Expect(machine.Name()).To(Equal("TestMachine"))
			Expect(machine.manager).To(Equal(manager))
		})
	})

	Describe("Machine", func() {
		BeforeEach(func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{}))
			machine, err = FromYAML("testdata/machine.yaml", manager)
		})

		Describe("Watchers", func() {
			It("Should return the machine watchers", func() {
				watchers := machine.Watchers()
				Expect(watchers).To(Equal(machine.WatcherDefs))
			})
		})

		Describe("Name", func() {
			It("Should return the name", func() {
				Expect(machine.Name()).To(Equal("TestMachine"))
			})
		})
	})

	Describe("Validate", func() {
		It("Should check common problems", func() {
			machine.MachineName = ""
			Expect(machine.Validate()).To(MatchError("a machine name is required"))

			machine.MachineName = "ginkgo"
			machine.MachineVersion = ""
			Expect(machine.Validate()).To(MatchError("a machine version is required"))

			machine.MachineVersion = "1.2.3"
			machine.InitialState = ""
			Expect(machine.Validate()).To(MatchError("an initial state is required"))

			machine.InitialState = "unknown"
			Expect(machine.Validate()).To(MatchError("no transitions defined"))

			machine.Transitions = []*Transition{{}}
			Expect(machine.Validate()).To(MatchError("no watchers defined"))

			machine.WatcherDefs = []*watchers.WatcherDef{{}}
			Expect(machine.Validate()).ToNot(HaveOccurred())
		})
	})

	Describe("Start", func() {
		It("Should start the machine using the manager", func() {
			wg := &sync.WaitGroup{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{}))
			machine, err = FromYAML("testdata/machine.yaml", manager)
			Expect(err).ToNot(HaveOccurred())
			manager.EXPECT().Run(gomock.AssignableToTypeOf(ctx), wg)

			machine.SplayStart = 0

			<-machine.Start(ctx, wg)
			Expect(machine.startTime.IsZero()).To(BeFalse())
		})
	})

	Describe("Stop", func() {
		It("Should not panic when nil", func() {
			machine.Stop()
		})

		It("Should stop a running machine", func() {
			machine.ctx, machine.cancel = context.WithCancel(context.Background())
			machine.startTime = time.Now()

			machine.Stop()
			Expect(machine.startTime.IsZero()).To(BeTrue())
			Expect(machine.ctx.Err()).To(HaveOccurred())
		})
	})

	Describe("State", func() {
		It("Should return the current state", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{}))
			machine, err = FromYAML("testdata/machine.yaml", manager)
			Expect(err).ToNot(HaveOccurred())
			machine.ctx = context.Background()
			Expect(machine.State()).To(Equal("unknown"))
		})
	})

	Describe("Transition", func() {
		It("Should initiate the event", func() {
			manager.EXPECT().SetMachine(gomock.AssignableToTypeOf(&Machine{}))
			machine, err = FromYAML("testdata/machine.yaml", manager)
			Expect(err).ToNot(HaveOccurred())
			machine.ctx = context.Background()
			machine.Transition("fire_1")
			Expect(machine.State()).To(Equal("one"))
			machine.RegisterNotifier(service)
			service.EXPECT().Warnf(machine, "machine", "Could not fire '%s' event while in %s", "fire_10", "one")
			machine.Transition("fire_10")
			Expect(machine.State()).To(Equal("one"))
		})
	})
})
