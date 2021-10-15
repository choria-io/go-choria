// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package compound

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/providers/data/ddl"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filter/Compound")
}

var _ = Describe("Compound", func() {
	var log *logrus.Entry

	BeforeEach(func() {
		log = logrus.NewEntry(logrus.New())
		log.Logger.SetOutput(GinkgoWriter)
	})

	Describe("MatchExprString", func() {
		It("Should correctly match", func() {
			query := [][]map[string]string{{{"expr": `with("foo") && scout("x").name=="bob" && scout("x").value==1`}}}
			df := ddl.FuncMap{
				"scout": {
					F: func(q string) interface{} {
						return map[string]interface{}{
							"name":  "bob",
							"value": 1,
						}
					},
					Name: "scout",
					DDL:  &ddl.DDL{Metadata: ddl.Metadata{Name: "scout", Timeout: 1}},
				},
			}

			match := MatchExprString(query, json.RawMessage{}, []string{"foo"}, []string{}, df, log)
			Expect(match).To(BeTrue())
		})
	})
})
