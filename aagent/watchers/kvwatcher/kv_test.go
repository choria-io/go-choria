// Copyright (c) 2021-2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package kvwatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/choria-io/go-choria/aagent/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/KvWatcher")
}

var _ = Describe("AAgent/Watchers/KvWatcher", func() {
	var (
		w       *Watcher
		machine *model.MockMachine
		mockctl *gomock.Controller
		kv      *MockKeyValue
		kve     *MockKeyValueEntry
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		machine = model.NewMockMachine(mockctl)
		machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		machine.EXPECT().Facts().Return(json.RawMessage(`{}`)).MinTimes(1)
		machine.EXPECT().Data().Return(map[string]any{}).MinTimes(1)
		machine.EXPECT().DataGet("machines").MinTimes(1)

		wi, err := New(machine, "kv", nil, nil, "", "", "1m", time.Hour, map[string]any{
			"bucket":        "PLUGINS",
			"key":           "machines",
			"mode":          "poll",
			"bucket_prefix": false,
		})
		Expect(err).ToNot(HaveOccurred())
		w = wi.(*Watcher)
		kv = NewMockKeyValue(mockctl)
		kve = NewMockKeyValueEntry(mockctl)
		kve.EXPECT().Revision().Return(uint64(1)).MinTimes(1)
		kv.EXPECT().Get("machines").Return(kve, nil)
		w.kv = kv
	})

	AfterEach(func() {
		mockctl.Finish()
	})

	Describe("Specification/Poll json parsing (#2037)", func() {
		It("Should handle a trailing newline", func() {
			kve.EXPECT().Value().Return([]byte("{\"spec\": \"foo\"}\n")).MinTimes(1)
			machine.EXPECT().DataPut("machines", map[string]any{"spec": "foo"}).Return(nil).Times(1)
			_, err := w.poll()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should handle a leading and trailing newline", func() {
			kve.EXPECT().Value().Return([]byte("\n{\"spec\": \"foo\"}\n")).MinTimes(1)
			machine.EXPECT().DataPut("machines", map[string]any{"spec": "foo"}).Return(nil).Times(1)
			_, err := w.poll()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should handle a leading and trailing unicode whitespace", func() {
			kve.EXPECT().Value().Return([]byte("\n   \t{\"spec\": \"foo\"}\t  \n")).MinTimes(1)
			machine.EXPECT().DataPut("machines", map[string]any{"spec": "foo"}).Return(nil).Times(1)
			_, err := w.poll()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
