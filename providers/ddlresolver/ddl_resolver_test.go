// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package ddlresolver

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMcoRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Providers/DDLRresolver")
}
