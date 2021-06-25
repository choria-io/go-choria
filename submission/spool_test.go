package submission

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSpool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spool")
}
