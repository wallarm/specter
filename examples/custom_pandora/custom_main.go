package main

import (
	"math/rand"
	"time"

	"github.com/spf13/afero"
	"github.com/wallarm/specter/cli"
	phttp "github.com/wallarm/specter/components/phttp/import"
	"github.com/wallarm/specter/core"
	coreimport "github.com/wallarm/specter/core/import"
	"github.com/wallarm/specter/core/register"
	"go.uber.org/zap"
)

type Ammo struct {
	URL        string
	QueryParam string
}

type Sample struct {
	URL              string
	ShootTimeSeconds float64
}

type GunConfig struct {
	Target string `validate:"required"` // Configuration will fail, without target defined
}

type Gun struct {
	// Configured on construction.
	conf GunConfig

	// Configured on Bind, before shooting.
	aggr core.Aggregator // May be your custom Aggregator.
	core.GunDeps
}

func NewGun(conf GunConfig) *Gun {
	return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	g.aggr = aggr
	g.GunDeps = deps
	return nil
}

func (g *Gun) Shoot(ammo core.Ammo) {
	customAmmo := ammo.(*Ammo) // Shoot will panic on unexpected ammo type. Panic cancels shooting.
	g.shoot(customAmmo)
}

func (g *Gun) shoot(ammo *Ammo) {
	start := time.Now()
	defer func() {
		g.aggr.Report(Sample{ammo.URL, time.Since(start).Seconds()})
	}()
	// Put your shoot logic here.
	g.Log.Info("Custom shoot", zap.String("target", g.conf.Target), zap.Any("ammo", ammo))
	time.Sleep(time.Duration(rand.Float64() * float64(time.Second)))
}

func main() {
	// Standard imports.
	fs := afero.NewOsFs()
	coreimport.Import(fs)

	// May not be imported, if you don't need http guns and etc.
	phttp.Import(fs)

	// Custom imports. Integrate your custom types into configuration system.
	coreimport.RegisterCustomJSONProvider("my-custom-provider-name", func() core.Ammo { return &Ammo{} })

	register.Gun("my-custom-gun-name", NewGun, func() GunConfig {
		return GunConfig{
			Target: "default target",
		}
	})
	register.Gun("my-custom/no-default", NewGun)

	cli.Run()
}
