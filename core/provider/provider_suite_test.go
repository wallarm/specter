package provider

import (
	"testing"

	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/lib/ginkgoutil"
)

func TestProvider(t *testing.T) {
	ginkgoutil.RunSuite(t, "AmmoQueue Suite")
}

func testDeps() core.ProviderDeps {
	return core.ProviderDeps{Log: ginkgoutil.NewLogger()}
}
