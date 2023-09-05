package monitoring

import (
	"expvar"
	"log"
	"strconv"

	"go.uber.org/atomic"
)

// TODO: use one rcrowley/go-metrics instead.

type Counter struct {
	i atomic.Int64
}

var _ expvar.Var = (*Counter)(nil)

func (c *Counter) String() string {
	return strconv.FormatInt(c.i.Load(), 10)
}

func (c *Counter) Add(delta int64) {
	c.i.Add(delta)
}

func (c *Counter) Set(value int64) {
	c.i.Store(value)
}

func (c *Counter) Get() int64 {
	return c.i.Load()
}

var counters = make(map[string]*Counter)

func NewCounter(name string) *Counter {
	if _, exists := counters[name]; exists {
		log.Printf("Counter with name %s already exists!", name)
		return counters[name]
	}

	v := &Counter{}
	expvar.Publish(name, v)
	counters[name] = v
	return v
}

func DropCounters() {
	counters = make(map[string]*Counter)
}
