// Copyright (c) 2020-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/ExecWatcher")
}

var _ = Describe("ExecWatcher", func() {
	var (
		mockctl     *gomock.Controller
		mockMachine *model.MockMachine
		watch       *Watcher
		now         time.Time
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())
		mockMachine = model.NewMockMachine(mockctl)

		mockMachine.EXPECT().Name().Return("file").AnyTimes()
		mockMachine.EXPECT().Identity().Return("ginkgo").AnyTimes()
		mockMachine.EXPECT().InstanceID().Return("1234567890").AnyTimes()
		mockMachine.EXPECT().Version().Return("1.0.0").AnyTimes()
		mockMachine.EXPECT().TimeStampSeconds().Return(now.Unix()).AnyTimes()
		mockMachine.EXPECT().Directory().Return(".").AnyTimes()

		now = time.Unix(1606924953, 0)

		wi, err := New(mockMachine, "ginkgo", []string{"always"}, nil, "fail", "success", "2m", time.Second, map[string]any{
			"path": filepath.Join("bin", "sh"),
		})
		Expect(err).ToNot(HaveOccurred())
		watch = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("setProperties", func() {
		It("Should parse valid properties", func() {
			prop := map[string]any{
				"path":                 "cmd",
				"gather_initial_state": "t",
			}
			Expect(watch.setProperties(prop)).ToNot(HaveOccurred())
			Expect(watch.properties.Path).To(Equal("cmd"))
			Expect(watch.properties.Initial).To(BeTrue())
		})

		It("Should handle errors", func() {
			watch.properties = &Properties{}
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("path is required"))
		})

		It("Should require a owner when managing content", func() {
			watch.properties = &Properties{
				Path:     "/some/file",
				Contents: "test",
			}
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("owner is required when managing content"))
		})

		It("Should require a group when managing content", func() {
			watch.properties = &Properties{
				Path:     "/some/file",
				Contents: "test",
				Owner:    "ginkgo",
			}
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("group is required when managing content"))
		})

		It("Should require a mode when managing content", func() {
			watch.properties = &Properties{
				Path:     "/some/file",
				Contents: "test",
				Owner:    "ginkgo",
				Group:    "ginkgo",
			}
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("mode is required when managing content"))
		})

		It("Should require a valid mode when managing content", func() {
			watch.properties = &Properties{
				Path:     "/some/file",
				Contents: "test",
				Owner:    "ginkgo",
				Group:    "ginkgo",
				Mode:     "foo",
			}
			err := watch.setProperties(map[string]any{})
			Expect(err).To(MatchError("invalid mode, must be a string like 0700"))
		})

		It("Should allow content to be managed", func() {
			watch.properties = &Properties{
				Path:     "/some/file",
				Contents: "test",
				Owner:    "root",
				Group:    "root",
				Mode:     "0700",
			}
			err := watch.setProperties(map[string]any{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("watch", func() {
		When("Managing content", func() {
			var (
				td  string
				f   string
				usr *user.User
				grp *user.Group
			)

			BeforeEach(func() {
				var err error

				usr, err = user.Current()
				Expect(err).ToNot(HaveOccurred())

				grp, err = user.LookupGroupId(usr.Gid)
				Expect(err).ToNot(HaveOccurred())

				mockMachine.EXPECT().State().Return("always").AnyTimes()
				mockMachine.EXPECT().Facts().Return([]byte("{}")).AnyTimes()
				mockMachine.EXPECT().Data().Return(map[string]any{
					"test":  "test_data",
					"group": grp.Name,
					"owner": usr.Name,
					"mode":  "0700",
				}).AnyTimes()

				td = GinkgoT().TempDir()
				f = filepath.Join(td, "the.file")

			})

			It("Should correctly manage the file", func() {
				watch.properties = &Properties{
					Path:     f,
					Contents: `{{ lookup "data.test" "default" | ToUpper }}`,
					Owner:    `{{ lookup "data.owner" "nobody"}}`,
					Group:    `{{ lookup "data.group" "nobody"}}`,
					Mode:     `{{ lookup "data.mode" "0000"}}`,
				}

				st, err := watch.watch()
				Expect(err).ToNot(HaveOccurred())
				Expect(st).To(Equal(Changed))

				st, err = watch.watch()
				Expect(err).ToNot(HaveOccurred())
				Expect(st).To(Equal(Unchanged))

				stat, err := os.Stat(watch.properties.Path)
				Expect(err).ToNot(HaveOccurred())
				b, err := os.ReadFile(watch.properties.Path)
				Expect(err).ToNot(HaveOccurred())

				Expect(stat.Mode()).To(Equal(os.FileMode(0700)))
				Expect(b).To(Equal([]byte("TEST_DATA")))

				Expect(os.Chmod(f, os.FileMode(0600))).To(Succeed())

				st, err = watch.watch()
				Expect(err).ToNot(HaveOccurred())
				Expect(st).To(Equal(Changed))

				st, err = watch.watch()
				Expect(err).ToNot(HaveOccurred())
				Expect(st).To(Equal(Unchanged))

				Expect(os.WriteFile(f, []byte("bad"), 0700)).To(Succeed())

				st, err = watch.watch()
				Expect(err).ToNot(HaveOccurred())
				Expect(st).To(Equal(Changed))

				st, err = watch.watch()
				Expect(err).ToNot(HaveOccurred())
				Expect(st).To(Equal(Unchanged))
			})
		})
	})

	Describe("CurrentState", func() {
		It("Should be a valid state", func() {
			watch.previous = Changed
			cs := watch.CurrentState()
			csj, err := cs.(*StateNotification).JSON()
			Expect(err).ToNot(HaveOccurred())

			event := map[string]any{}
			err = json.Unmarshal(csj, &event)
			Expect(err).ToNot(HaveOccurred())
			delete(event, "id")

			Expect(event).To(Equal(map[string]any{
				"time":            "2020-12-02T16:02:33Z",
				"type":            "io.choria.machine.watcher.file.v1.state",
				"subject":         "ginkgo",
				"specversion":     "1.0",
				"source":          "io.choria.machine",
				"datacontenttype": "application/json",
				"data": map[string]any{
					"id":               "1234567890",
					"identity":         "ginkgo",
					"machine":          "file",
					"name":             "ginkgo",
					"protocol":         "io.choria.machine.watcher.file.v1.state",
					"type":             "file",
					"version":          "1.0.0",
					"timestamp":        float64(now.Unix()),
					"previous_outcome": "changed",
					"path":             filepath.Join("bin", "sh"),
				},
			}))
		})
	})
})
