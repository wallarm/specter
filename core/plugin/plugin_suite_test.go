package plugin

import (
	"testing"

	"github.com/wallarm/specter/lib/ginkgoutil"
)

func TestPlugin(t *testing.T) {
	ginkgoutil.RunSuite(t, "Plugin Suite")
}
