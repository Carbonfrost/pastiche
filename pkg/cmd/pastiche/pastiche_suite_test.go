package pastiche_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPastiche(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pastiche Suite")
}
