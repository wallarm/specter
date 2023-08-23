package provider

import (
	"context"

	"github.com/wallarm/specter/core"
)

// NewNum returns dummy provider, that provides 0, 1 .. n int sequence as ammo.
// May be useful for test or in when Gun don't need ammo.
func NewNum(limit int) core.Provider {
	return &num{
		limit: limit,
		sink:  make(chan core.Ammo),
	}
}

func NewNumBuffered(limit int) core.Provider {
	return &num{
		limit: limit,
		sink:  make(chan core.Ammo, limit),
	}
}

type NumConfig struct {
	Limit int
}

func NewNumConf(conf NumConfig) core.Provider {
	return NewNum(conf.Limit)
}

type num struct {
	i     int
	limit int
	sink  chan core.Ammo
}

func (n *num) Run(ctx context.Context, _ core.ProviderDeps) error {
	defer close(n.sink)
	for ; n.limit <= 0 || n.i < n.limit; n.i++ {
		select {
		case n.sink <- n.i:
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

func (n *num) Acquire() (a core.Ammo, ok bool) {
	a, ok = <-n.sink
	return
}

func (n *num) Release(core.Ammo) {}
