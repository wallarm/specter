package phttp

import (
	"testing"

	"github.com/wallarm/specter/lib/ginkgoutil"
)

func TestPhttp(t *testing.T) {
	ginkgoutil.RunSuite(t, "HTTP Suite")
}
