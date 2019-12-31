package opa

import (
	"context"
	"io/ioutil"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"
	"github.com/sirupsen/logrus"
)

func TestFileSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Opa")
}

var _ = Describe("Opa", func() {
	var log *logrus.Entry

	BeforeEach(func() {
		log = logrus.NewEntry(logrus.New())
		log.Logger.SetOutput(GinkgoWriter)
	})

	Describe("Evaluate", func() {
		It("Should support basic evaluations", func() {
			inputs := map[string]interface{}{"hello": "world"}
			e, err := New("io.choria.ginkgo", "data.io.choria.ginkgo.allow", Logger(log), File("testdata/test1.rego"), Trace())
			Expect(err).ToNot(HaveOccurred())

			pass, err := e.Evaluate(context.Background(), inputs)
			Expect(err).ToNot(HaveOccurred())
			Expect(pass).To(BeTrue())
		})

		It("Should support functions", func() {
			inputs := map[string]interface{}{"hello": "world"}
			ran := 0
			f := rego.FunctionDyn(&rego.Function{
				Name: "ginkgo",
				Decl: types.NewFunction(types.Args(), types.B),
			},
				func(_ rego.BuiltinContext, _ []*ast.Term) (*ast.Term, error) {
					ran++
					return ast.BooleanTerm(true), nil
				})

			e, err := New("io.choria.ginkgo", "data.io.choria.ginkgo.allow", Logger(log), File("testdata/func1.rego"), Trace(), Function(f))
			Expect(err).ToNot(HaveOccurred())

			pass, err := e.Evaluate(context.Background(), inputs)
			Expect(err).ToNot(HaveOccurred())
			Expect(pass).To(BeTrue())
			Expect(ran).To(Equal(1))
		})

		It("Should support supplied policies", func() {
			inputs := map[string]interface{}{"hello": "world"}
			policy, err := ioutil.ReadFile("testdata/test1.rego")
			Expect(err).ToNot(HaveOccurred())

			e, err := New("io.choria.ginkgo", "data.io.choria.ginkgo.allow", Logger(log), Policy(policy), Trace())
			Expect(err).ToNot(HaveOccurred())

			pass, err := e.Evaluate(context.Background(), inputs)
			Expect(err).ToNot(HaveOccurred())
			Expect(pass).To(BeTrue())
		})

		It("Should run the same with multiple inputs", func() {
			inputs := map[string]interface{}{"hello": "world"}
			e, err := New("io.choria.ginkgo", "data.io.choria.ginkgo.allow", Logger(log), File("testdata/test1.rego"), Trace())
			Expect(err).ToNot(HaveOccurred())

			pass, err := e.Evaluate(context.Background(), inputs)
			Expect(err).ToNot(HaveOccurred())
			Expect(pass).To(BeTrue())

			inputs["hello"] = "foo"
			pass, err = e.Evaluate(context.Background(), inputs)
			Expect(err).ToNot(HaveOccurred())
			Expect(pass).To(BeFalse())

			inputs["hello"] = "world"
			pass, err = e.Evaluate(context.Background(), inputs)
			Expect(err).ToNot(HaveOccurred())
			Expect(pass).To(BeTrue())
		})
	})
})
