// Copyright (c) 2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package fs

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal/FS")
}

var _ = Describe("FS", func() {
	Describe("JSON Files", func() {
		It("Should have valid JSON files", func() {
			err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if filepath.Ext(path) != ".json" {
					return nil
				}

				d := map[string]any{}
				jd, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				return json.Unmarshal(jd, &d)
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
