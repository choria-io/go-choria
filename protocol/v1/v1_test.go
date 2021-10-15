// Copyright (c) 2017-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestV1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Protocol/V1")
}
