package aggregator

import (
	"context"

	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/coreutil"
)

// NewDiscard returns Aggregator that just throws reported ammo away.
func NewDiscard() core.Aggregator {
	return discard{}
}

type discard struct{}

func (discard) Run(ctx context.Context, _ core.AggregatorDeps) error {
	<-ctx.Done()
	return nil
}

func (discard) Report(s core.Sample) {
	coreutil.ReturnSampleIfBorrowed(s)
}
