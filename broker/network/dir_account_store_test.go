package network

import (
	"io/ioutil"

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

		das, err = newDirAccountStore(notificationReceiver, "testdata/accounts/nats/choria_operator")
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
			Expect(err).To(MatchError("no matching JWT found for missing"))
		})

		It("Should load the correct JWT", func() {
			d, err := das.Fetch("ACLORPCUGYF7SE3ZGZ7NJ4RG4NVKGSTV325P57JCXIOPOHOYWLQUHCUN")
			Expect(err).ToNot(HaveOccurred())
			expected, err := ioutil.ReadFile("testdata/accounts/nats/choria_operator/accounts/choria_account/choria_account.jwt")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(string(expected)))
		})
	})
})
