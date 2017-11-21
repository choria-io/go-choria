package choria

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type tc struct {
	KeyOne  string `confkey:"test.one" default:"one" environment:"ONE_OVERRIDE"`
	KeyTwo  string `confkey:"test.two" default:"two"`
	BoolKey bool   `confkey:"test.bool" default:"true"`
}

func TestMCollective(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCollective")
}

var _ = Describe("NewChoria", func() {
	It("Should initialize choria correctly", func() {
		c := newChoria()
		Expect(c.DiscoveryHost).To(Equal("puppet"))
		Expect(c.DiscoveryPort).To(Equal(8085))
		Expect(c.UseSRVRecords).To(BeTrue())
	})
})

var _ = Describe("NewConfig", func() {
	It("Should correctly parse config files", func() {
		c, err := NewConfig("testdata/choria.cfg")
		Expect(err).ToNot(HaveOccurred())

		Expect(c.Choria.DiscoveryHost).To(Equal("pdb.example.com"))
		Expect(c.Registration).To(Equal("Foo"))
		Expect(c.RegisterInterval).To(Equal(10))
		Expect(c.RegistrationSplay).To(BeTrue())
		Expect(c.Collectives).To(Equal([]string{"c_1", "c_2", "c_3"}))
		Expect(c.MainCollective).To(Equal("c_1"))
		Expect(c.KeepLogs).To(Equal(5))
		Expect(c.LibDir).To(Equal([]string{"/dir1", "/dir2", "/dir3", "/dir4"}))
		Expect(c.DefaultDiscoveryOptions).To(Equal([]string{"one", "two"}))
		Expect(c.Choria.RandomizeMiddlewareHosts).To(BeTrue())
	})
})

var _ = Describe("setDefaults", func() {
	It("Should set the right defaults", func() {
		data := tc{}

		Expect(data.KeyOne).To(BeEmpty())
		Expect(data.BoolKey).To(BeFalse())

		setDefaults(&data)

		Expect(data.KeyOne).To(Equal("one"))
		Expect(data.KeyTwo).To(Equal("two"))
		Expect(data.BoolKey).To(BeTrue())
	})
})

var _ = Describe("itemWithKey", func() {
	It("Should find the right key", func() {
		data := tc{}

		k, err := itemWithKey(data, "test.one")
		Expect(err).ToNot(HaveOccurred())
		Expect(k).To(Equal("KeyOne"))

		k, err = itemWithKey(data, "test.two")
		Expect(err).ToNot(HaveOccurred())
		Expect(k).To(Equal("KeyTwo"))
	})

	It("Should fail for unknown keys", func() {
		data := tc{}

		k, err := itemWithKey(data, "test.three")
		Expect(k).To(BeEmpty())
		Expect(err).To(MatchError("Can't find any structure element that holds test.three"))
	})
})

var _ = Describe("setItemWithKey", func() {
	It("Should seet the right item", func() {
		data := tc{}

		setItemWithKey(&data, "test.one", "new value")
		Expect(data.KeyOne).To(Equal("new value"))
		Expect(data.KeyTwo).To(BeEmpty())
	})

	// race condition with when the environment gets set, skipping
	// It("Should support environment overrides", func() {
	// 	data := tc{}
	// 	setItemWithKey(&data, "test.one", "new value")
	// 	Expect(data.KeyOne).To(Equal("new value"))

	// 	os.Setenv("ONE_OVERRIDE", "OVERRIDE")
	// 	for {
	// 		if _, ok := os.LookupEnv("ONE_OVERRIDE"); ok {
	// 			break
	// 		}
	// 	}

	// 	setItemWithKey(&data, "test.one", "new value")
	// 	Expect(data.KeyOne).To(Equal("OVERRIDE"))
	// })
})

var _ = Describe("tag", func() {
	It("Should get the right tag", func() {
		data := tc{}

		t, success := tag(data, "KeyOne", "default")
		Expect(success).To(BeTrue())
		Expect(t).To(Equal("one"))

		t, success = tag(data, "Fail", "default")
		Expect(success).To(BeFalse())
		Expect(t).To(BeEmpty())
	})
})
