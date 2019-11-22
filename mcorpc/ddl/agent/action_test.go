package agent

import (
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("McoRPC/DDL/Agent/Action", func() {
	var pkg *DDL
	var err error
	var act Action

	BeforeEach(func() {
		act = Action{
			Input: map[string]*ActionInputItem{
				"int":     &ActionInputItem{Type: "integer", Optional: true, Default: 1},
				"float":   &ActionInputItem{Type: "float", Optional: true},
				"number":  &ActionInputItem{Type: "number", Optional: true},
				"string":  &ActionInputItem{Type: "string", MaxLength: 20, Optional: false},
				"boolean": &ActionInputItem{Type: "boolean", Optional: true},
				"list":    &ActionInputItem{Type: "list", Enum: []string{"one", "two"}, Optional: true},
				"hash":    &ActionInputItem{Type: "Hash", Optional: true},
				"array":   &ActionInputItem{Type: "Array", Optional: true},
			},
			Output: map[string]*ActionOutputItem{
				"int":    &ActionOutputItem{Type: "integer", Default: 1},
				"string": &ActionOutputItem{Type: "string"},
			},
		}

		pkg, err = New(path.Join("testdata", "mcollective", "agent", "package.json"))
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("ValidateAndConvertToDDLTypes", func() {
		It("Should correctly convert inputs", func() {
			orig := map[string]string{
				"int":    "10",
				"float":  "100.2",
				"number": "10.1",
				"string": "hello world",
				"list":   "one",
				"hash":   `{"hello":"world"}`,
				"array":  `["hello", "world"]`,
			}

			converted, warnings, err := act.ValidateAndConvertToDDLTypes(orig)
			Expect(warnings).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
			Expect(converted["int"].(int64)).To(Equal(int64(10)))
			Expect(converted["float"].(float64)).To(Equal(100.2))
			Expect(converted["number"].(float64)).To(Equal(10.1))
			Expect(converted["string"].(string)).To(Equal("hello world"))
			Expect(converted["list"].(string)).To(Equal("one"))
			Expect(converted["hash"]).To(Equal(map[string]interface{}{"hello": "world"}))
			Expect(converted["array"]).To(Equal([]interface{}{"hello", "world"}))
		})

		It("Should validate inputs", func() {
			orig := map[string]string{
				"string": "123456789012345678901234567890",
			}
			_, warnings, err := act.ValidateAndConvertToDDLTypes(orig)
			Expect(warnings).To(HaveLen(0))
			Expect(err).To(MatchError("invalid value for 'string': is longer than 20 characters"))
		})

		It("Should check for missing inputs", func() {
			orig := map[string]string{
				"int": "1",
			}
			_, warnings, err := act.ValidateAndConvertToDDLTypes(orig)
			Expect(warnings).To(HaveLen(0))
			Expect(err).To(MatchError("input 'string' is required"))
		})

		It("Should set defaults", func() {
			orig := map[string]string{
				"float":  "100.2",
				"number": "10.1",
				"string": "hello world",
				"list":   "one",
			}

			act.Input["boolean"].Default = false
			act.Input["boolean"].Optional = true

			res, warnings, err := act.ValidateAndConvertToDDLTypes(orig)
			Expect(warnings).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKey("int"))
			Expect(res["int"].(int)).To(Equal(1))
			Expect(res).To(HaveKey("boolean"))
			Expect(res["boolean"].(bool)).To(BeFalse())
		})
	})

	Describe("ValidateInputString", func() {
		It("Should correctly validate the input string as its correct type", func() {
			err := act.ValidateInputString("int", "hello world")
			Expect(err).To(MatchError("'hello world' is not a valid integer"))
			err = act.ValidateInputString("int", "10")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("ValidateInputValue", func() {
		It("Should validate integer", func() {
			warnings, err := act.ValidateInputValue("int", 10)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("int", "a")
			Expect(err).To(MatchError("is not an integer"))
			Expect(warnings).To(HaveLen(0))
		})

		It("Should validate number", func() {
			warnings, err := act.ValidateInputValue("number", 10)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("number", 10.2)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("number", "a")
			Expect(err).To(MatchError("is not a number"))
			Expect(warnings).To(HaveLen(0))
		})

		It("Should validate float", func() {
			warnings, err := act.ValidateInputValue("float", 10.2)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("float", "a")
			Expect(err).To(MatchError("is not a float"))
			Expect(warnings).To(HaveLen(0))
		})

		It("Should validate string", func() {
			warnings, err := act.ValidateInputValue("string", "hello world")
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("string", "123456789012345678901234567890")
			Expect(err).To(MatchError("is longer than 20 characters"))
			Expect(warnings).To(HaveLen(0))

			act.Input["string"].MaxLength = 0
			warnings, err = act.ValidateInputValue("string", "123456789012345678901234567890")
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))
		})

		It("Should validate boolean", func() {
			warnings, err := act.ValidateInputValue("boolean", true)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("boolean", false)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("boolean", "foo")
			Expect(err).To(MatchError("is not a boolean"))
			Expect(warnings).To(HaveLen(0))
		})

		It("Should validate list", func() {
			warnings, err := act.ValidateInputValue("list", "one")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("list", "two")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(HaveLen(0))

			warnings, err = act.ValidateInputValue("list", "three")
			Expect(err).To(MatchError("should be one of one, two"))

			warnings, err = act.ValidateInputValue("list", false)
			Expect(err).To(MatchError("should be a string"))
		})

		It("Should validate unknowns", func() {
			act.Input["unkn"] = &ActionInputItem{Type: "unknown"}

			warnings, err := act.ValidateInputValue("unkn", "one")
			Expect(warnings).To(HaveLen(0))
			Expect(err).To(MatchError("unsupported input type 'unknown'"))

			warnings, err = act.ValidateInputValue("invalid", "one")
			Expect(warnings).To(HaveLen(0))
			Expect(err).To(MatchError("unknown input 'invalid'"))
		})

		It("Should validate string content", func() {
			act.Input["string"].Validation = "^bob"
			warnings, err := act.ValidateInputValue("string", "hello world")
			Expect(err).To(MatchError("input does not match '^bob'"))
			Expect(warnings).To(HaveLen(0))

			act.Input["string"].Validation = "bob"
			warnings, err = act.ValidateInputValue("string", "hello world")
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(Equal([]string{"Unsupported validator 'bob'"}))
		})

		It("Should validate hash content", func() {
			warnings, err := act.ValidateInputValue("hash", "hello world")
			Expect(warnings).To(HaveLen(0))
			Expect(err).To(MatchError("is not a hash map"))

			warnings, err = act.ValidateInputValue("hash", map[string]string{"hello": "world"})
			Expect(warnings).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should validate array content", func() {
			warnings, err := act.ValidateInputValue("array", "hello world")
			Expect(warnings).To(HaveLen(0))
			Expect(err).To(MatchError("is not an array"))

			warnings, err = act.ValidateInputValue("array", []string{"hello", "world"})
			Expect(warnings).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("SetOutputDefaults", func() {
		It("Should set defaults correctly", func() {
			res := map[string]interface{}{}
			act.SetOutputDefaults(res)
			Expect(res["int"].(int)).To(Equal(1))
			Expect(res).ToNot(HaveKey("string"))
		})
	})

	Describe("OutputNames", func() {
		It("Should retrieve the output names", func() {
			Expect(act.OutputNames()).To(Equal([]string{"int", "string"}))
		})
	})

	Describe("InputNames", func() {
		It("Should retrieve the input names", func() {
			install, err := pkg.ActionInterface("install")
			Expect(err).ToNot(HaveOccurred())
			Expect(install.InputNames()).To(Equal([]string{"package", "version"}))
		})

		It("Should support actions with no inputs", func() {
			cu, err := pkg.ActionInterface("apt_checkupdates")
			Expect(err).ToNot(HaveOccurred())
			Expect(cu.InputNames()).To(Equal([]string{}))
		})
	})

	Describe("ValidateRequestData", func() {
		It("Should support empty input lists", func() {
			cu, err := pkg.ActionInterface("apt_checkupdates")
			Expect(err).ToNot(HaveOccurred())

			w, err := cu.ValidateRequestData(map[string]interface{}{})
			Expect(err).ToNot(HaveOccurred())
			Expect(w).To(HaveLen(0))
		})

		It("Should handle required inputs", func() {
			install, err := pkg.ActionInterface("install")
			Expect(err).ToNot(HaveOccurred())

			w, err := install.ValidateRequestData(map[string]interface{}{})
			Expect(err).To(MatchError("input 'package' is required"))
			Expect(w).To(HaveLen(0))
		})

		It("Should handle actions with no inputs but inputs being recieved", func() {
			cu, err := pkg.ActionInterface("apt_checkupdates")
			Expect(err).ToNot(HaveOccurred())

			w, err := cu.ValidateRequestData(map[string]interface{}{"test":"test"})
			Expect(err).To(MatchError("request contains inputs while none are declared in the DDL"))
			Expect(w).To(HaveLen(0))
		})

		It("Should detect extra inputs", func() {
			install, err := pkg.ActionInterface("install")
			Expect(err).ToNot(HaveOccurred())

			w, err := install.ValidateRequestData(map[string]interface{}{
				"package":"zsh",
				"other":"test",
			})
			Expect(err).To(MatchError("request contains an input 'other' that is not declared in the DDL. Valid inputs are: package, version"))
			Expect(w).To(HaveLen(0))
		})
	})

	Describe("RequiresInput", func() {
		It("Should correctly report require state", func() {
			install, err := pkg.ActionInterface("install")
			Expect(err).ToNot(HaveOccurred())

			Expect(install.RequiresInput("package")).To(BeTrue())
			Expect(install.RequiresInput("version")).To(BeFalse())
		})
	})

	Describe("validateStringValidation", func() {
		It("Should support shellsafe", func() {
			w, err := validateStringValidation("shellsafe", ">")
			Expect(w).To(HaveLen(0))
			Expect(err).To(MatchError("may not contain '>'"))

			w, err = validateStringValidation("shellsafe", "foo")
			Expect(w).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should suport ipv4address", func() {
			w, err := validateStringValidation("ipv4address", "2a00:1450:4002:807::200e")
			Expect(w).To(HaveLen(0))
			Expect(err).To(MatchError("2a00:1450:4002:807::200e is not an IPv4 address"))

			w, err = validateStringValidation("ipv4address", "1.1.1.1")
			Expect(w).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support ipv6address", func() {
			w, err := validateStringValidation("ipv6address", "2a00:1450:4002:807::200e")
			Expect(w).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())

			w, err = validateStringValidation("ipv6address", "1.1.1.1")
			Expect(w).To(HaveLen(0))
			Expect(err).To(MatchError("1.1.1.1 is not an IPv6 address"))
		})

		It("Should support ipaddress", func() {
			w, err := validateStringValidation("ipaddress", "2a00:1450:4002:807::200e")
			Expect(w).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())

			w, err = validateStringValidation("ipaddress", "bob")
			Expect(w).To(HaveLen(0))
			Expect(err).To(MatchError("bob is not an IP address"))
		})

		It("Should warn but not fail for validators that start with alpha characters", func() {
			w, err := validateStringValidation("foo", "2a00:1450:4002:807::200e")
			Expect(w).To(HaveLen(1))
			Expect(w[0]).To(Equal("Unsupported validator 'foo'"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should support regex validators", func() {
			w, err := validateStringValidation("^2a00", "2a00:1450:4002:807::200e")
			Expect(w).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())

			w, err = validateStringValidation("\\d+", "bob")
			Expect(w).To(HaveLen(0))
			Expect(err).To(MatchError("input does not match '\\d+'"))
		})
	})
})
