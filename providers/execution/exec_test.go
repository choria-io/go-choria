// Copyright (c) 2025, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	iu "github.com/choria-io/go-choria/internal/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGovernor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Execution")
}

var _ = Describe("Execution", func() {
	var (
		td    string
		jobId string
		p     *Process
		err   error
	)

	BeforeEach(func() {
		jobId = iu.UniqueID()
		td = GinkgoT().TempDir()

		p, err = New("ginkgo", "agent", "action", iu.UniqueID(), "ginkgo.example.net", jobId, "echo", []string{"hello", "world"}, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("New", func() {
		It("Should validate the input", func() {
			_, err := New("", "X", "x", "", "x", "x", "x", nil, nil)
			Expect(err).To(MatchError(ContainSubstring("no caller")))

			_, err = New("x", "X", "x", "", "x", "x", "x", nil, nil)
			Expect(err).To(MatchError(ContainSubstring("no request id")))

			_, err = New("x", "X", "x", "x", "x", "", "x", nil, nil)
			Expect(err).To(MatchError(ContainSubstring("no id")))

			_, err = New("x", "X", "x", "x", "", "x", "x", nil, nil)
			Expect(err).To(MatchError(ContainSubstring("no identity")))

			_, err = New("x", "X", "x", "x", "x", "x", "", nil, nil)
			Expect(err).To(MatchError(ContainSubstring("no command")))
		})
	})

	Describe("IsMatch", func() {
		It("Should match", func() {
			Expect(p.IsMatch(&ListQuery{})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{Action: "action"})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Action: "other"})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{Agent: "agent"})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Agent: "other"})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{Before: time.Now().Add(-time.Hour)})).To(BeFalse())
			Expect(p.IsMatch(&ListQuery{Before: time.Now().Add(time.Hour)})).To(BeTrue())

			Expect(p.IsMatch(&ListQuery{Caller: "ginkgo"})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Caller: "other"})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{Command: "echo"})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Command: "other"})).To(BeFalse())

			p.PidFile = filepath.Join(td, "pid")
			Expect(p.IsMatch(&ListQuery{Running: true})).To(BeFalse())
			Expect(os.WriteFile(p.PidFile, []byte(strconv.Itoa(os.Getpid())), 0700)).To(Succeed())
			Expect(p.IsMatch(&ListQuery{Running: true})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Completed: true})).To(BeFalse())
			Expect(p.IsMatch(&ListQuery{Completed: true})).To(BeFalse())
			Expect(os.WriteFile(p.PidFile, []byte("0"), 0700)).To(Succeed())
			Expect(p.IsMatch(&ListQuery{Completed: true})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Running: true})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{Identity: "ginkgo.example.net"})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{Identity: "other"})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{RequestID: p.RequestID})).To(BeTrue())
			Expect(p.IsMatch(&ListQuery{RequestID: "other"})).To(BeFalse())

			Expect(p.IsMatch(&ListQuery{Since: time.Now()})).To(BeFalse())
			Expect(p.IsMatch(&ListQuery{Since: time.Now().Add(-time.Hour)})).To(BeTrue())

			Expect(p.IsMatch(&ListQuery{
				Identity: "ginkgo.example.net",
				Caller:   "ginkgo",
			})).To(BeTrue())

			Expect(p.IsMatch(&ListQuery{
				Identity: "ginkgo.example.net",
				Caller:   "other",
			})).To(BeFalse())
		})
	})

	Describe("Load", func() {
		It("Should load the correct process", func() {
			_, err := Load(td, "x")
			Expect(err).To(MatchError(ErrSpecificationNotFound))

			_, err = p.CreateSpool(td)
			Expect(err).ToNot(HaveOccurred())

			lp, err := Load(td, jobId)
			Expect(err).ToNot(HaveOccurred())

			Expect(lp).To(Equal(p))
		})
	})

	Describe("CreateSpool", func() {
		It("Should create a spool", func() {
			j, err := p.CreateSpool(td)
			Expect(err).ToNot(HaveOccurred())

			Expect(filepath.Join(td, jobId, "spec.json")).To(BeAnExistingFile())

			f, err := os.ReadFile(filepath.Join(td, jobId, "spec.json"))
			Expect(err).ToNot(HaveOccurred())
			Expect(json.RawMessage(f)).To(Equal(j))
		})

		It("Should detect duplicates", func() {
			_, err = p.CreateSpool(td)
			Expect(err).ToNot(HaveOccurred())
			_, err = p.CreateSpool(td)
			Expect(err).To(MatchError(ErrDuplicateJob))
		})
	})

	Describe("HasStarted", func() {
		It("Should correctly detect if the process was started", func() {
			Expect(p.PidFile).To(BeEmpty())
			Expect(p.HasStarted()).To(BeFalse())

			p.PidFile = filepath.Join(td, "pid")
			Expect(p.HasStarted()).To(BeFalse())

			Expect(os.WriteFile(p.PidFile, []byte("10"), 0700)).To(Succeed())
			Expect(p.HasStarted()).To(BeTrue())
		})
	})

	Describe("IsRunning", func() {
		It("Should correctly detect if the process is running", func() {
			p.PidFile = filepath.Join(td, "pid")

			Expect(os.WriteFile(p.PidFile, []byte(strconv.Itoa(os.Getpid())), 0700)).To(Succeed())
			Expect(p.IsRunning()).To(BeTrue())

			Expect(os.WriteFile(p.PidFile, []byte("0"), 0700)).To(Succeed())
			Expect(p.IsRunning()).To(BeFalse())
		})
	})

	Describe("Stderr", func() {
		It("Should handle missing paths", func() {
			_, err := p.Stderr()
			Expect(err).To(MatchError(ErrInvalidProcess))
		})

		It("Should read the file", func() {
			p.StderrFile = filepath.Join(td, "stderr")
			body := []byte("stderr")
			err = os.WriteFile(p.StderrFile, body, 0700)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Stderr()).To(Equal(body))
		})
	})

	Describe("Stdout", func() {
		It("Should handle missing paths", func() {
			_, err := p.Stdout()
			Expect(err).To(MatchError(ErrInvalidProcess))
		})

		It("Should read the file", func() {
			p.StdoutFile = filepath.Join(td, "stdout")
			body := []byte("stdout")
			err = os.WriteFile(p.StdoutFile, body, 0700)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Stdout()).To(Equal(body))
		})
	})

	Describe("ParsePid", func() {
		It("Should detect no pid file set", func() {
			pid, err := p.ParsePid()
			Expect(pid).To(Equal(-1))
			Expect(err).To(MatchError(ContainSubstring("no pid file configured")))
		})

		It("Should handle read errors", func() {
			p.PidFile = filepath.Join(td, "pid")
			pid, err := p.ParsePid()
			Expect(pid).To(Equal(-1))
			if runtime.GOOS == "windows" {
				Expect(err).To(MatchError(ContainSubstring("cannot find the file specified")))
			} else {
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			}
		})

		It("Should handle empty pid files", func() {
			p.PidFile = filepath.Join(td, "pid")
			f, err := os.Create(p.PidFile)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			pid, err := p.ParsePid()
			Expect(pid).To(Equal(-1))
			Expect(err).To(MatchError(ContainSubstring("0 length pid")))
		})

		It("Should handle corrupt pid files", func() {
			p.PidFile = filepath.Join(td, "pid")
			err := os.WriteFile(p.PidFile, []byte("a"), 0700)
			Expect(err).ToNot(HaveOccurred())
			pid, err := p.ParsePid()
			Expect(pid).To(Equal(-1))
			Expect(err).To(MatchError(ContainSubstring("invalid syntax")))
		})

		It("Should correctly parse the pid file", func() {
			p.PidFile = filepath.Join(td, "pid")
			err := os.WriteFile(p.PidFile, []byte("10"), 0700)
			Expect(err).ToNot(HaveOccurred())
			pid, err := p.ParsePid()
			Expect(pid).To(Equal(10))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
