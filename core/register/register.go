package register

import (
	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/plugin"
)

func RegisterPtr(ptr interface{}, name string, newPlugin interface{}, defaultConfigOptional ...interface{}) {
	plugin.Register(plugin.PtrType(ptr), name, newPlugin, defaultConfigOptional...)
}

func Provider(name string, newProvider interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Provider
	RegisterPtr(ptr, name, newProvider, defaultConfigOptional...)
}

func Limiter(name string, newLimiter interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Schedule
	RegisterPtr(ptr, name, newLimiter, defaultConfigOptional...)
}

func Gun(name string, newGun interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Gun
	RegisterPtr(ptr, name, newGun, defaultConfigOptional...)
}

func Aggregator(name string, newAggregator interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Aggregator
	RegisterPtr(ptr, name, newAggregator, defaultConfigOptional...)
}

func DataSource(name string, newDataSource interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.DataSource
	RegisterPtr(ptr, name, newDataSource, defaultConfigOptional...)
}

func DataSink(name string, newDataSink interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.DataSink
	RegisterPtr(ptr, name, newDataSink, defaultConfigOptional...)
}
