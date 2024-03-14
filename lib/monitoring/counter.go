package monitoring

import (
	//"expvar"
	//"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type Counter struct {
	prometheusCounter prometheus.Counter
}

//var _ expvar.Var = (*Counter)(nil)

//func (c *Counter) String() string {
//	return strconv.FormatInt(c.i.Load(), 10)
//}

func (c *Counter) Add(delta float64) {
	c.prometheusCounter.Add(delta)
}

//func (c *Counter) Set(value int64) {
//	c.i.Store(value)
//}

func (c *Counter) Get() float64 {
	// В Prometheus нет прямого способа получить текущее значение Counter
	// Вместо этого обычно используются метрики для экспорта данных
	return 0 // заглушка
}

var counters = make(map[string]*Counter)

func NewCounter(name string) *Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: "Counter help message",
	})
	prometheus.MustRegister(counter)
	return &Counter{prometheusCounter: counter}
}

func DropCounters() {
	counters = make(map[string]*Counter)
}

//metricsRegistry := metrics.NewRegistry()
//prometheusClient := prometheusmetrics.NewPrometheusProvider(metrics.DefaultRegistry, "whatever", "something", prometheus.DefaultRegisterer, 1*time.Second)
