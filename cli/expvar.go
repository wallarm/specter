package cli

import (
	"time"

	"github.com/wallarm/specter/core/engine"
	"github.com/wallarm/specter/lib/monitoring"
)

func newEngineMetrics() engine.Metrics {
	monitoring.DropCounters()
	return engine.Metrics{
		Request:        monitoring.NewCounter("engine_Requests"),
		Response:       monitoring.NewCounter("engine_Responses"),
		InstanceStart:  monitoring.NewCounter("engine_UsersStarted"),
		InstanceFinish: monitoring.NewCounter("engine_UsersFinished"),
	}
}

func startReport(m engine.Metrics) {
	evReqPS := monitoring.NewCounter("engine_ReqPS")
	evResPS := monitoring.NewCounter("engine_ResPS")
	evActiveUsers := monitoring.NewCounter("engine_ActiveUsers")
	evActiveRequests := monitoring.NewCounter("engine_ActiveRequests")

	requests := m.Request.Get()
	responses := m.Response.Get()
	go func() {
		var requestsNew, responsesNew int64
		// TODO: there is no guarantee, that we will run exactly after 1 second.
		// So, when we get 1 sec +-10ms, we getting 990-1010 calculate intervals and +-2% RPS in reports.
		// Consider using rcrowley/go-metrics.Meter.
		for range time.NewTicker(1 * time.Second).C {
			requestsNew = m.Request.Get()
			responsesNew = m.Response.Get()
			rps := responsesNew - responses
			requestPerSecond := requestsNew - requests
			activeUsers := m.InstanceStart.Get() - m.InstanceFinish.Get()
			activeRequests := requestsNew - responsesNew
			//zap.S().Infof(
			//	"[ENGINE] %d resp/s; %d req/s; %d users; %d active\n",
			//	rps, requestPerSecond, activeUsers, activeRequests)

			requests = requestsNew
			responses = responsesNew

			evActiveUsers.Set(activeUsers)
			evActiveRequests.Set(activeRequests)
			evReqPS.Set(requestPerSecond)
			evResPS.Set(rps)
		}
	}()
}
