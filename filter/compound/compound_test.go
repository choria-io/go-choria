// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package compound

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
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
			cases := []struct {
				query  string
				expect bool
			}{
				{`semver(scout("x").version, "= 1.2.3")`, true},
				{`semver(scout("x").version, "< 1.2.4")`, true},
				{`semver(scout("x").version, "> 1.0.0")`, true},
				{`semver(scout("x").version, "< 1.0.0")`, false},
				{`with("foo") && scout("x").name=="bob" && scout("x").value==1`, true},
			}

			for _, tc := range cases {
				query := [][]map[string]string{{{"expr": tc.query}}}
				df := ddl.FuncMap{
					"scout": {
						F: func(q string) interface{} {
							return map[string]interface{}{
								"name":    "bob",
								"value":   1,
								"version": "1.2.3",
							}
						},
						Name: "scout",
						DDL:  &ddl.DDL{Metadata: ddl.Metadata{Name: "scout", Timeout: 1}},
					},
				}

				match := MatchExprString(query, json.RawMessage{}, []string{"foo"}, []string{}, df, log)
				Expect(match).To(Equal(tc.expect))
			}
		})
	})
})
