// Copyright (c) 2023, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddl

import (
	"encoding/json"
	"testing"

	"github.com/choria-io/go-choria/internal/fs"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFileContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generators/DDL")
}

var _ = Describe("Agents", func() {
	Describe("ValidateSchemaFromFS", func() {
		It("Should handle any data type", func() {
			ddlBytes, err := fs.FS.ReadFile("ddl/cache/agent/scout.json")
			Expect(err).ToNot(HaveOccurred())

			var addl agent.DDL
			err = json.Unmarshal(ddlBytes, &addl)
			Expect(err).ToNot(HaveOccurred())

			g := &Generator{}
			err = g.ValidateJSON(&addl)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
