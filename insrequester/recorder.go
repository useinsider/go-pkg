package insrequester

import (
	"github.com/slok/goresilience/metrics"
	"time"
)

type recorder struct {
	metrics.Recorder
	requester *Request
}

func (r *recorder) WithID(id string) metrics.Recorder {
	return r
}

func (r *recorder) ObserveCommandExecution(start time.Time, success bool) {
	r.Recorder.ObserveCommandExecution(start, success)
}

func (r *recorder) IncRetry() {
	if r.requester.onRetry != nil {
		r.requester.onRetry()
	}
	r.Recorder.IncRetry()
}

func (r *recorder) IncTimeout() {
	if r.requester.onTimeout != nil {
		r.requester.onTimeout()
	}
	r.Recorder.IncTimeout()
}

func (r *recorder) IncBulkheadQueued() {
	r.Recorder.IncBulkheadQueued()
}

func (r *recorder) IncBulkheadProcessed() {
	r.Recorder.IncBulkheadProcessed()
}

func (r *recorder) IncCircuitbreakerState(state string) {
	if r.requester.onCircuitBreakerStateChange != nil {
		r.requester.onCircuitBreakerStateChange(r.requester, r.requester.cbState, CBState(state))
	}

	if state == string(CBStateOpen) {
		if r.requester.onCircularBreakerOpen != nil {
			r.requester.onCircularBreakerOpen()
		}
	} else if state == string(CBStateHalfOpen) {
		if r.requester.onCircularBreakerHalfOpen != nil {
			r.requester.onCircularBreakerHalfOpen()
		}
	} else if state == string(CBStateClosed) {
		if r.requester.onCircularBreakerClosed != nil {
			r.requester.onCircularBreakerClosed()
		}
	}

	r.requester.cbState = CBState(state)
	r.Recorder.IncCircuitbreakerState(state)
}

func (r *recorder) IncBulkheadTimeout() {
	r.Recorder.IncBulkheadTimeout()
}

func (r *recorder) IncChaosInjectedFailure(kind string) {
	r.Recorder.IncChaosInjectedFailure(kind)
}

func (r *recorder) SetConcurrencyLimitInflightExecutions(q int) {
	r.Recorder.SetConcurrencyLimitInflightExecutions(q)
}

func NewRecorder(requester *Request) metrics.Recorder {
	return &recorder{
		Recorder:  metrics.Dummy,
		requester: requester,
	}
}
