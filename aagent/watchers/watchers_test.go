package watchers

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/build"
)

func TestWatchers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aagent/Watchers")
}

var _ = Describe("Aagent/Watchers", func() {
	var (
		mockctl  *gomock.Controller
		machine  *MockMachine
		watcherC *MockWatcherConstructor
		watcher  *MockWatcher
		manager  *Manager
		err      error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		machine = NewMockMachine(mockctl)
		watcherC = NewMockWatcherConstructor(mockctl)
		watcher = NewMockWatcher(mockctl)
		manager = New()
		manager.machine = machine
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("SetMachine", func() {
		It("Should set the machine", func() {
			err = manager.SetMachine(1)
			Expect(err).To(MatchError("supplied machine does not implement watchers.Machine"))
			err = manager.SetMachine(machine)
			Expect(manager.machine).To(Equal(machine))
		})
	})

	Describe("Plugins", func() {
		BeforeEach(func() {
			build.MachineWatchers = []string{}
			plugins = nil
			watcherC.EXPECT().Type().Return("mock").AnyTimes()
		})

		It("Should register watchers", func() {
			err = RegisterWatcherPlugin("mock watcher version 1", watcherC)
			Expect(err).ToNot(HaveOccurred())
			Expect(plugins).To(Equal(map[string]WatcherConstructor{
				"mock": watcherC,
			}))
			Expect(build.MachineWatchers).To(Equal([]string{"mock watcher version 1"}))
		})
	})

	Describe("configureWatchers", func() {
		BeforeEach(func() {
			watcherC.EXPECT().Type().Return("mock").AnyTimes()
			plugins = nil
			watcherC.EXPECT().Type().Return("mock").AnyTimes()
			err = RegisterWatcherPlugin("mock watcher version 1", watcherC)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support registered watchers", func() {
			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any())
			machine.EXPECT().Directory().Return(filepath.Dir(".")).AnyTimes()
			machine.EXPECT().Watchers().Return([]*WatcherDef{
				{
					Name:              "mwatcher",
					Type:              "mock",
					StateMatch:        []string{"one"},
					FailTransition:    "failed",
					SuccessTransition: "passed",
					Interval:          "1m",
					AnnounceDuration:  0,
					Properties: map[string]interface{}{
						"path": "/dev/null",
					},
				},
			})

			watcher.EXPECT().Name().Return("mwatcher").AnyTimes()
			watcherC.EXPECT().New(machine, "mwatcher", []string{"one"}, "failed", "passed", "1m", 0*time.Second, map[string]interface{}{
				"path": "/dev/null",
			}).Return(watcher, nil).AnyTimes()

			err = manager.configureWatchers()
			Expect(err).ToNot(HaveOccurred())

			w, ok := manager.watchers["mwatcher"]
			Expect(ok).To(BeTrue())
			Expect(w).To(Equal(watcher))
		})
	})
})
