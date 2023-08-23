package netsample

import (
	"testing"

	"github.com/wallarm/specter/lib/ginkgoutil"
)

func TestNetsample(t *testing.T) {
	ginkgoutil.RunSuite(t, "Netsample Suite")
}
