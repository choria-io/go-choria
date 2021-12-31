// Copyright (c) 2020-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package confkey

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Confkey")
}

type TestData struct {
	PlainString string        `confkey:"plain_string" validate:"shellsafe"`
	CommaSplit  []string      `confkey:"comma_split" type:"comma_split"`
	PathSplit   []string      `confkey:"path_split" type:"path_split"`
	ColonSplit  []string      `confkey:"colon_split" type:"colon_split"`
	StringEnum  string        `confkey:"loglevel" validate:"enum=debug,info,warn" default:"warn"`
	Int         int           `confkey:"int"`
	Int64       int64         `confkey:"int64"`
	TitleString string        `confkey:"title_string" type:"title_string"`
	PathString  string        `confkey:"path_string" type:"path_string"`
	Bool        bool          `confkey:"bool"`
	T           time.Duration `confkey:"interval" type:"duration" default:"1h" environment:"T"`
	Doc         string        `confkey:"doc" url:"http://example.com" deprecated:"1" description:"hello world"`
	NonCK       string
}

var _ = Describe("Confkey", func() {
	var d TestData

	BeforeEach(func() {
		d = TestData{}
	})

	Describe("Type", func() {
		It("Should get the right value", func() {
			ds, ok := Type(&d, "doc")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("string"))

			ds, ok = Type(&d, "colon_split")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("colon_split"))
		})
	})

	Describe("IsDeprecated", func() {
		It("Should get the right value", func() {
			ds, ok := IsDeprecated(d, "doc")
			Expect(ok).To(BeTrue())
			Expect(ds).To(BeTrue())

			ds, ok = IsDeprecated(d, "bool")
			Expect(ok).To(BeFalse())
			Expect(ds).To(BeFalse())
		})
	})

	Describe("URL", func() {
		It("Should get the right value", func() {
			ds, ok := URL(d, "doc")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("http://example.com"))
		})
	})

	Describe("Environment", func() {
		It("Should get the right value", func() {
			ds, ok := Environment(d, "interval")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("T"))
		})
	})

	Describe("DefaultString", func() {
		It("Should get the right value", func() {
			ds, ok := DefaultString(d, "loglevel")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("warn"))
		})
	})

	Describe("Validation", func() {
		It("Should get the right value", func() {
			ds, ok := Validation(d, "loglevel")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("enum=debug,info,warn"))
		})
	})

	Describe("Description", func() {
		It("Should get the right value", func() {
			ds, ok := Description(d, "doc")
			Expect(ok).To(BeTrue())
			Expect(ds).To(Equal("hello world"))
		})
	})

	Describe("KeyTag", func() {
		It("Should get the right data", func() {
			t, ok := KeyTag(d, "loglevel", "validate")
			Expect(ok).To(BeTrue())
			Expect(t).To(Equal("enum=debug,info,warn"))

			t, ok = KeyTag(d, "bool", "validate")
			Expect(ok).To(BeFalse())
			Expect(t).To(Equal(""))

			t, ok = KeyTag(d, "na", "validate")
			Expect(ok).To(BeFalse())
			Expect(t).To(Equal(""))
		})
	})

	Describe("FindFields", func() {
		It("Should find al the right keys", func() {
			found, err := FindFields(d, "split|bool")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(Equal(strings.Fields("bool colon_split comma_split path_split")))
		})
	})

	Describe("FieldWithKey", func() {
		It("Should find the right key", func() {
			k, err := FieldWithKey(d, "bool")
			Expect(k).To(Equal("Bool"))
			Expect(err).ToNot(HaveOccurred())

			k, err = FieldWithKey(d, "nonexisting")
			Expect(k).To(Equal(""))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("InterfaceWithKey", func() {
		It("Should retrieve the right value", func() {
			d.Int64 = 10
			untyped, ok := InterfaceWithKey(&d, "int64")
			Expect(ok).To(BeTrue())

			_, ok = InterfaceWithKey(&d, "foo")
			Expect(ok).To(BeFalse())

			i64, ok := untyped.(int64)
			if !ok {
				Fail(fmt.Sprintf("Retrieved data is not int64: <%v %v>", reflect.TypeOf(untyped), untyped))
			}

			Expect(i64).To(Equal(int64(10)))
		})
	})

	Describe("Int64WithKey", func() {
		It("Should get the right int64", func() {
			Expect(Int64WithKey(&d, "int64")).To(Equal(int64(0)))
			d.Int64 = 10
			Expect(Int64WithKey(&d, "int64")).To(Equal(int64(10)))
		})

		It("Should be 0 when not found", func() {
			Expect(Int64WithKey(&d, "unknown")).To(Equal(int64(0)))
		})
	})

	Describe("IntWithKey", func() {
		It("Should get the right int", func() {
			Expect(IntWithKey(&d, "int")).To(Equal(0))
			d.Int = 10
			Expect(IntWithKey(&d, "int")).To(Equal(10))
		})

		It("Should be 0 when not found", func() {
			Expect(IntWithKey(&d, "unknown")).To(Equal(0))
		})
	})

	Describe("BoolWithKey", func() {
		It("Should get the right bool", func() {
			d.Bool = false
			Expect(BoolWithKey(&d, "bool")).To(BeFalse())
			d.Bool = true
			Expect(BoolWithKey(&d, "bool")).To(BeTrue())
		})

		It("Should be false when not found", func() {
			Expect(BoolWithKey(&d, "unknown")).To(BeFalse())
		})
	})

	Describe("StringListWithKey", func() {
		It("Should get the right list", func() {
			d.CommaSplit = []string{"one", "two"}
			Expect(StringListWithKey(&d, "comma_split")).To(Equal([]string{"one", "two"}))
		})

		It("Should be empty when not found", func() {
			Expect(StringListWithKey(&d, "unknown")).To(Equal([]string{}))
		})
	})

	Describe("StringFieldWithKey", func() {
		It("Should get the right string", func() {
			d.StringEnum = "warn"
			Expect(StringFieldWithKey(&d, "loglevel")).To(Equal("warn"))
		})
		It("Should be empty when not found", func() {
			Expect(StringFieldWithKey(&d, "foo")).To(Equal(""))
			Expect(StringFieldWithKey(&d, "loglevel")).To(Equal(""))
		})
	})

	Describe("Validate", func() {
		It("Should validate the struct", func() {
			err := Validate(TestData{PlainString: "un > safe"})
			Expect(err).To(MatchError("PlainString shellsafe validation failed: may not contain '>'"))
		})
	})

	Describe("SetStructDefaults", func() {
		It("Should set defaults", func() {
			err := SetStructDefaults(d)
			Expect(err).To(MatchError("pointer is required"))

			err = SetStructDefaults(&d)
			Expect(err).ToNot(HaveOccurred())
			Expect(d.StringEnum).To(Equal("warn"))
			Expect(d.PlainString).To(Equal(""))
			Expect(d.T).To(Equal(time.Hour))
		})
	})

	Describe("SetStructFieldWithKey", func() {
		It("Should set and validate the field", func() {
			err := SetStructFieldWithKey(&d, "plain_string", "hello world")
			Expect(err).ToNot(HaveOccurred())

			err = SetStructFieldWithKey(&d, "plain_string", "un > safe")
			Expect(err).To(MatchError("PlainString shellsafe validation failed: may not contain '>'"))
		})

		It("Should handle unknown fields", func() {
			err := SetStructFieldWithKey(&d, "missing", "hello world")
			Expect(err).To(MatchError("can't find any structure element configured with confkey 'missing'"))
		})

		It("Should support comma_split", func() {
			err := SetStructFieldWithKey(&d, "comma_split", "foo, bar, baz")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.CommaSplit).To(Equal([]string{"foo", "bar", "baz"}))

			err = SetStructFieldWithKey(&d, "comma_split", "foo")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.CommaSplit).To(Equal([]string{"foo"}))

		})

		It("Should support colon_split", func() {
			err := SetStructFieldWithKey(&d, "colon_split", "/foo:/bar:/baz")

			Expect(err).ToNot(HaveOccurred())
			Expect(d.ColonSplit).To(Equal([]string{"/foo", "/bar", "/baz"}))
		})

		It("Should support path_split", func() {
			var err error

			if runtime.GOOS == "windows" {
				err = SetStructFieldWithKey(&d, "path_split", "/foo;/bar;/baz")
			} else {
				err = SetStructFieldWithKey(&d, "path_split", "/foo:/bar:/baz")
			}

			Expect(err).ToNot(HaveOccurred())
			Expect(d.PathSplit).To(Equal([]string{"/foo", "/bar", "/baz"}))
		})

		It("Should support ints", func() {
			err := SetStructFieldWithKey(&d, "int", "1")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.Int).To(Equal(1))
		})

		It("Should support title_string", func() {
			err := SetStructFieldWithKey(&d, "title_string", "foobar")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.TitleString).To(Equal("Foobar"))
		})

		It("Should support path_string", func() {
			err := os.Setenv("HOME", "/home/joeuser")
			Expect(err).ToNot(HaveOccurred())
			err = os.Setenv("HOMEDRIVE", "C:\\")
			Expect(err).ToNot(HaveOccurred())
			err = os.Setenv("HOMEDIR", "myhome")
			Expect(err).ToNot(HaveOccurred())

			if runtime.GOOS == "windows" {
				err = SetStructFieldWithKey(&d, "path_string", "~\\ssl_dir")
			} else {
				err = SetStructFieldWithKey(&d, "path_string", "~/ssl_dir")
			}
			Expect(err).ToNot(HaveOccurred())
			if runtime.GOOS == "windows" {
				Expect(d.PathString).To(Equal("C:\\myhome\\ssl_dir"))
			} else {
				Expect(d.PathString).To(Equal("/home/joeuser/ssl_dir"))
			}
		})

		It("Should support bools", func() {
			for _, v := range []string{"1", "YES", "y", "tRue", "t"} {
				err := SetStructFieldWithKey(&d, "bool", v)
				Expect(err).ToNot(HaveOccurred())
				Expect(d.Bool).To(Equal(true))
			}

			for _, v := range []string{"0", "NO", "f", "FalSE", "n", "invalid"} {
				err := SetStructFieldWithKey(&d, "bool", v)
				Expect(err).ToNot(HaveOccurred())
				Expect(d.Bool).To(Equal(false))
			}
		})

		It("Should support durations", func() {
			err := SetStructFieldWithKey(&d, "interval", "1s")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.T).To(Equal(1 * time.Second))

			err = SetStructFieldWithKey(&d, "interval", "10")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.T).To(Equal(10 * time.Second))

			err = SetStructFieldWithKey(&d, "interval", "1h")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.T).To(Equal(1 * time.Hour))
		})
	})
})
