package ammo

import (
	"net/http"

	phttp "github.com/wallarm/specter/components/guns/http"
	"github.com/wallarm/specter/core/aggregator/netsample"
)

type Request interface {
	http.Request
}

var _ phttp.Ammo = (*GunAmmo)(nil)

type GunAmmo struct {
	req       *http.Request
	id        uint64
	tag       string
	isInvalid bool
}

func (g GunAmmo) Request() (*http.Request, *netsample.Sample) {
	sample := netsample.Acquire(g.tag)
	sample.SetID(g.id)
	return g.req, sample
}

func (g GunAmmo) ID() uint64 {
	return g.id
}

func (g GunAmmo) IsInvalid() bool {
	return g.isInvalid
}

func NewGunAmmo(req *http.Request, tag string, id uint64) GunAmmo {
	return GunAmmo{
		req: req,
		id:  id,
		tag: tag,
	}
}
