package watchers

import (
	"path/filepath"
	"testing"

	"github.com/choria-io/go-choria/aagent/watchers/execwatcher"
	"github.com/choria-io/go-choria/aagent/watchers/filewatcher"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWatchers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aagent/Watchers")
}

var _ = Describe("Aagent/Watchers", func() {
	var (
		mockctl *gomock.Controller
		machine *MockMachine
		manager *Manager
		err     error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		machine = NewMockMachine(mockctl)
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

	Describe("configureWatchers", func() {
		It("Should support file watchers", func() {
			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any())
			machine.EXPECT().Directory().Return(filepath.Dir(".")).AnyTimes()
			machine.EXPECT().Watchers().Return([]*WatcherDef{
				&WatcherDef{
					Name:              "fwatcher",
					Type:              "file",
					StateMatch:        []string{"one"},
					FailTransition:    "failed",
					SuccessTransition: "passed",
					Interval:          "1m",
					announceDuration:  0,
					Properties: map[string]interface{}{
						"path": "/dev/null",
					},
				},
			})

			err := manager.configureWatchers()
			Expect(err).ToNot(HaveOccurred())

			w, ok := manager.watchers["fwatcher"]
			Expect(ok).To(BeTrue())

			Expect(w).To(BeAssignableToTypeOf(&filewatcher.Watcher{}))
			Expect(w.Name()).To(Equal("fwatcher"))
			Expect(w.Type()).To(Equal("file"))
		})

		It("Should support exec watchers", func() {
			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any())
			machine.EXPECT().Watchers().Return([]*WatcherDef{
				&WatcherDef{
					Type:              "exec",
					Name:              "execwatcher",
					StateMatch:        []string{"one"},
					FailTransition:    "failed",
					SuccessTransition: "passed",
					Interval:          "1m",
					announceDuration:  0,
					Properties: map[string]interface{}{
						"command": "/dev/null",
					},
				},
			})

			err := manager.configureWatchers()
			Expect(err).ToNot(HaveOccurred())

			w, ok := manager.watchers["execwatcher"]
			Expect(ok).To(BeTrue())

			Expect(w).To(BeAssignableToTypeOf(&execwatcher.Watcher{}))
			Expect(w.Name()).To(Equal("execwatcher"))
			Expect(w.Type()).To(Equal("exec"))
		})

		It("Should handle unknown watchers", func() {
			machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any())
			machine.EXPECT().Watchers().Return([]*WatcherDef{
				&WatcherDef{
					Type: "other",
					Name: "otherwatcher",
				},
			})

			err := manager.configureWatchers()
			Expect(err).To(MatchError("unknown watcher 'other'"))
		})
	})
})
