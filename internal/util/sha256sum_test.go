// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"os"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Internal/Util/Sha256sums", func() {
	var (
		log *logrus.Entry
	)

	BeforeEach(func() {
		log = logrus.NewEntry(logrus.New())
		log.Logger.SetOutput(GinkgoWriter)
	})

	Describe("Sha256ChecksumDir", func() {
		It("Should create valid checksums", func() {
			if runtime.GOOS == "windows" {
				Skip("Tests not supported on windows")
			}

			expected, err := os.ReadFile("testdata/SHA256SUM.dir")
			Expect(err).ToNot(HaveOccurred())

			res, err := Sha256ChecksumDir("testdata/dir")
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(expected))
		})
	})

	Describe("Sha256VerifyDir", func() {
		It("Should verify valid files", func() {
			lines := 0
			ok, err := Sha256VerifyDir("testdata/SHA256SUM.good", "", log, func(f string, ok bool) {
				lines++
				if !ok {
					Fail(fmt.Sprintf("File %v did not match", f))
				}
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(lines).To(Equal(2))
		})

		It("Should detect bad files", func() {
			lines := 0
			ok, err := Sha256VerifyDir("testdata/SHA256SUM.bad", "testdata", log, func(f string, ok bool) {
				lines++
				switch f {
				case "9f28c.txt", "other/missing":
					if ok {
						Fail(fmt.Sprintf("File %s did not fail", f))
					}
				default:
					if !ok {
						Fail(fmt.Sprintf("File %v did not match", f))
					}
				}
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(lines).To(Equal(3))
		})

		It("Should handle corrupt sum files", func() {
			ok, err := Sha256VerifyDir("testdata/SHA256SUM.corrupt", "testdata", log, func(string, bool) {})
			Expect(err).To(MatchError("invalid sums file: malformed line 0"))
			Expect(ok).To(BeFalse())
		})
	})
})
