// Copyright (c) 2019-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package classes

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filter/Classes")
}

var _ = Describe("Classes", func() {
	var log *logrus.Entry

	BeforeEach(func() {
		log = logrus.WithFields(logrus.Fields{"testing": true})
		logrus.SetLevel(logrus.PanicLevel)
	})

	It("Should handle missing classes files", func() {
		Expect(MatchFile([]string{"x"}, "testdata/nonexisting.txt", log)).To(BeFalse())
	})

	It("Should support regex", func() {
		Expect(MatchFile([]string{"/test/"}, "testdata/classes.txt", log)).To(BeTrue())
		Expect(MatchFile([]string{"/TeSt/"}, "testdata/classes.txt", log)).To(BeTrue())
		Expect(MatchFile([]string{"/nonxisting/"}, "testdata/classes.txt", log)).To(BeFalse())
		Expect(MatchFile([]string{"/NoNxIsTiNg/"}, "testdata/classes.txt", log)).To(BeFalse())
	})

	It("Should support exact matches", func() {
		Expect(MatchFile([]string{"role::testing"}, "testdata/classes.txt", log)).To(BeTrue())
		Expect(MatchFile([]string{"RoLe::TeStInG"}, "testdata/classes.txt", log)).To(BeTrue())
		Expect(MatchFile([]string{"nonxisting"}, "testdata/classes.txt", log)).To(BeFalse())
		Expect(MatchFile([]string{"NoNxIsTInG"}, "testdata/classes.txt", log)).To(BeFalse())
	})
})
