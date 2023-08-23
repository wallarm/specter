package phttp

import (
	"net/http"

	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/aggregator/netsample"
)

//go:generate mockery --name=Ammo --case=underscore --outpkg=ammomock

// Ammo ammo interface for http based guns.
// http ammo providers should produce ammo that implements Ammo.
// http guns should use convert ammo to Ammo, not to specific implementation.
// Returned request have
type Ammo interface {
	// TODO: instead of sample use it wrapper with httptrace and more usable interface.
	Request() (*http.Request, *netsample.Sample)
	// Id unique ammo id. Usually equals to ammo num got from provider.
	ID() uint64
	IsInvalid() bool
}

type Gun interface {
	Shoot(ammo Ammo)
	Bind(sample netsample.Aggregator, deps core.GunDeps) error
}

func WrapGun(g Gun) core.Gun {
	if g == nil {
		return nil
	}
	return &gunWrapper{g}
}

type gunWrapper struct{ Gun }

func (g *gunWrapper) Shoot(ammo core.Ammo) {
	g.Gun.Shoot(ammo.(Ammo))
}

func (g *gunWrapper) Bind(a core.Aggregator, deps core.GunDeps) error {
	return g.Gun.Bind(netsample.UnwrapAggregator(a), deps)
}
