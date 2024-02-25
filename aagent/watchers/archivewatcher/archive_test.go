// Copyright (c) 2021-2024, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/choria-io/go-choria/aagent/model"
)

func TestMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAgent/Watchers/MachinesWatcher")
}

var _ = Describe("AAgent/Watchers/MachinesWatcher", func() {
	var (
		w       *Watcher
		machine *model.MockMachine
		mockctl *gomock.Controller
		td      string
		err     error
	)

	BeforeEach(func() {
		if runtime.GOOS == "windows" {
			Skip("not supported on windows")
		}

		td, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		mockctl = gomock.NewController(GinkgoT())

		machine = model.NewMockMachine(mockctl)
		machine.EXPECT().Directory().Return(td).AnyTimes()
		machine.EXPECT().Infof(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		machine.EXPECT().Warnf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		machine.EXPECT().Errorf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		machine.EXPECT().Facts().Return(json.RawMessage("{}")).AnyTimes()
		machine.EXPECT().Data().Return(map[string]any{}).AnyTimes()

		wi, err := New(machine, "machines", nil, "", "", "1m", time.Hour, map[string]any{
			"source":   "https://example.net",
			"creates":  "testdata/creates",
			"target":   td,
			"checksum": "x",
		})
		Expect(err).ToNot(HaveOccurred())
		w = wi.(*Watcher)
	})

	AfterEach(func() {
		mockctl.Finish()
		os.RemoveAll(td)
	})

	Describe("mkTempDir", func() {
		It("Should create a temp dir above the creates dir in tmp", func() {
			md, err := os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(md).ToNot(BeEmpty())
			defer os.RemoveAll(md)

			w.properties.TargetDirectory = filepath.Join(md, "machines")
			w.properties.Creates = "machine-name"

			td, err := w.mkTempDir()
			Expect(err).ToNot(HaveOccurred())
			Expect(td).ToNot(BeEmpty())
			Expect(filepath.Dir(td)).To(Equal(filepath.Join(w.properties.TargetDirectory, "tmp")))
		})
	})

	Describe("verify", func() {
		BeforeEach(func() {
			w.properties.ContentChecksums = "SHA256SUMS"
			w.properties.ContentChecksumsChecksum = "40cb790b7199be45f3116354f87b2bdc3aa520a1eb056aa3608911cf40d1f821"
		})

		It("Should handle bad templates", func() {
			w.properties.ContentChecksumsChecksum = "{{bad}}"
			err := w.verify("testdata/good")
			Expect(err).To(MatchError("could not parse template on verify_checksum property"))

			w.properties.ContentChecksumsChecksum = `{{lookup "x" ""}}`
			err = w.verify("testdata/good")
			Expect(err).To(MatchError("verify_checksum template resulted in an empty string"))
		})

		It("Should process templates", func() {
			w.properties.ContentChecksumsChecksum = `{{lookup "x" "40cb790b7199be45f3116354f87b2bdc3aa520a1eb056aa3608911cf40d1f821"}}`
			Expect(w.verify("testdata/good")).To(Succeed())

		})

		It("Should handle bad checksums", func() {
			w.properties.ContentChecksumsChecksum = "x"
			Expect(w.verify("testdata/good")).To(MatchError("checksum file SHA256SUMS has an invalid checksum \"40cb790b7199be45f3116354f87b2bdc3aa520a1eb056aa3608911cf40d1f821\" != \"x\""))
		})
	})

	Describe("verifyCreates", func() {
		BeforeEach(func() {
			w.properties.TargetDirectory = "testdata"
			w.properties.ContentChecksums = "SHA256SUMS"
			w.properties.ContentChecksumsChecksum = "40cb790b7199be45f3116354f87b2bdc3aa520a1eb056aa3608911cf40d1f821"
		})

		It("Should handle missing creates dir", func() {
			w.properties.Creates = "missing"
			creates, state, err := w.verifyCreates()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(MissingCreates))
			Expect(creates).To(Equal("testdata/missing"))
		})

		It("Should handle missing checksums", func() {
			w.properties.Creates = "incomplete"
			creates, state, err := w.verifyCreates()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(MissingChecksums))
			Expect(creates).To(Equal("testdata/incomplete"))
		})

		It("Should detect bad states", func() {
			w.properties.Creates = "bad"
			creates, state, err := w.verifyCreates()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(VerifyFailed))
			Expect(creates).To(Equal("testdata/bad"))
		})

		It("Should detect good states", func() {
			w.properties.Creates = "good"
			creates, state, err := w.verifyCreates()
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(VerifiedOK))
			Expect(creates).To(Equal("testdata/good"))
		})
	})
})
