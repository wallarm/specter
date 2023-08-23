package engine

import (
	"testing"

	"github.com/wallarm/specter/lib/ginkgoutil"
	"github.com/wallarm/specter/lib/monitoring"
)

func TestEngine(t *testing.T) {
	ginkgoutil.RunSuite(t, "Engine Suite")
}

func newTestMetrics() Metrics {
	return Metrics{
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
	}
}
