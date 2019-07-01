package network

import (
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("dirAccountStore", func() {
	var (
		mockctl              *gomock.Controller
		notificationReceiver accountNotificationReceiver
		das                  *dirAccountStore
		err                  error
	)

	BeforeEach(func() {
		mockctl = gomock.NewController(GinkgoT())

		das, err = newDirAccountStore(notificationReceiver, "testdata/accounts")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		das.Stop()
		mockctl.Finish()
	})

	Describe("Fetch", func() {
		It("Should handle missing files", func() {
			d, err := das.Fetch("missing")
			Expect(d).To(Equal(""))
			Expect(err).To(MatchError("could not retrieve account 'missing' from testdata/accounts/missing.jwt: open testdata/accounts/missing.jwt: no such file or directory"))
		})

		It("Should load the correct JWT", func() {
			d, err := das.Fetch("exists")
			Expect(d).To(Equal("jwt"))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
