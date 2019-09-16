package agent

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "McoRPC/DDL/Agent")
}

var _ = Describe("McoRPC/DDL/Agent", func() {
	var pkg *DDL
	var err error

	BeforeEach(func() {
		pkg, err = New(path.Join("testdata", "mcollective", "agent", "package.json"))
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("New", func() {
		It("Should fail for missing json files", func() {
			d, err := New(path.Join("testdata", "missing.json"))
			Expect(err.Error()).To(MatchRegexp("could not load DDL data: open.+missing.json"))
			Expect(d).To(BeNil())
		})

		It("Should fail for invalid json files", func() {
			d, err := New(path.Join("testdata", "invalid.json"))
			Expect(err).To(MatchError("could not parse JSON data in testdata/invalid.json: unexpected end of JSON input"))
			Expect(d).To(BeNil())
		})

		It("Should correctly load valid DDL files", func() {
			Expect(pkg.Metadata.Author).To(Equal("R.I.Pienaar <rip@devco.net>"))
			Expect(pkg.Metadata.Description).To(Equal("Manage Operating System Packages"))
			Expect(pkg.Metadata.License).To(Equal("Apache-2.0"))
			Expect(pkg.Metadata.Name).To(Equal("package"))
			Expect(pkg.Metadata.Timeout).To(Equal(180))
			Expect(pkg.Metadata.URL).To(Equal("https://github.com/choria-plugins/package-agent"))
			Expect(pkg.Metadata.Version).To(Equal("5.0.0"))
			Expect(pkg.Actions[3].Aggregation[0].Function).To(Equal("summary"))
		})
	})

	Describe("EachFile", func() {
		It("Should call the cb with the right files", func() {
			files := make(map[string]string)

			EachFile([]string{"nonexisting", "testdata"}, func(n, p string) bool {
				files[n] = p
				return false
			})

			Expect(files["package"]).To(Equal(filepath.Join("testdata", "mcollective", "agent", "package.json")))
		})
	})

	Describe("ActionList", func() {
		It("Should return the correct list", func() {
			Expect(pkg.ActionNames()).To(Equal([]string{"apt_checkupdates", "apt_update", "checkupdates", "count", "install", "md5", "purge", "status", "uninstall", "update", "yum_checkupdates", "yum_clean"}))
		})
	})

	Describe("HaveAction", func() {
		It("Should correctly determine if actions exist", func() {
			Expect(pkg.HaveAction("foo")).To(BeFalse())
			Expect(pkg.HaveAction("apt_checkupdates")).To(BeTrue())
		})
	})

	Describe("Timeout", func() {
		It("Should handle 0 second timeouts as 10 seconds", func() {
			pkg.Metadata.Timeout = 0

			Expect(pkg.Timeout()).To(Equal(time.Duration(10 * time.Second)))
		})

		It("Should handle timeouts correctly", func() {
			Expect(pkg.Timeout()).To(Equal(time.Duration(180 * time.Second)))
		})
	})

	Describe("ActionInterface", func() {
		It("Should retrieve the correct interface", func() {
			act, err := pkg.ActionInterface("install")
			Expect(err).ToNot(HaveOccurred())

			Expect(act.Name).To(Equal("install"))
			Expect(act.Description).To(Equal("Install a package"))
			Expect(act.Display).To(Equal("failed"))
			Expect(act.Output).To(HaveLen(8))
			Expect(act.Input).To(HaveLen(2))
		})

		It("Should handle unknown interfaces", func() {
			act, err := pkg.ActionInterface("unknown")
			Expect(err).To(HaveOccurred())
			Expect(act).To(BeNil())
		})
	})

	Describe("ToRuby", func() {
		It("Should generate ruby ddls", func() {
			out, err := pkg.ToRuby()
			Expect(err).ToNot(HaveOccurred())
			Expect(out).ToNot(HaveLen(0))
		})
	})

	Describe("AggregateResultJSON", func() {
		type reply struct {
			Statuscode mcorpc.StatusCode `json:"statuscode"`
			Statusmsg  string            `json:"statusmsg"`
			Data       json.RawMessage   `json:"data"`
		}

		It("Should aggregate the JSON data", func() {
			var replies []reply
			dat, err := ioutil.ReadFile("testdata/package_replies.json")
			Expect(err).ToNot(HaveOccurred())

			err = json.Unmarshal(dat, &replies)
			Expect(err).ToNot(HaveOccurred())

			act, err := pkg.ActionInterface("status")
			Expect(err).ToNot(HaveOccurred())

			for _, reply := range replies {
				err = act.AggregateResultJSON(reply.Data)
				Expect(err).ToNot(HaveOccurred())
			}

			summary, err := act.AggregateSummaryStrings()
			Expect(err).ToNot(HaveOccurred())
			Expect(summary).To(Equal(map[string]map[string]string{
				"arch": map[string]string{
					"x86_64": "6",
				},
				"ensure": map[string]string{
					"5.0.2-33.el7": "5",
					"5.0.2-31.el7": "1",
				},
			}))

			formatted, err := act.AggregateSummaryFormattedStrings()
			Expect(err).ToNot(HaveOccurred())
			Expect(formatted).To(Equal(map[string][]string{
				"ensure": []string{
					"5.0.2-33.el7: 5",
					"5.0.2-31.el7: 1",
				},
				"arch": []string{
					"x86_64: 6",
				},
			}))
		})
	})
})
