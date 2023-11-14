package insrequester

func (r *Request) OnCircularBreakerOpen(f func()) *Request {
	return r
}

func (r *Request) OnCircularBreakerHalfOpen(f func()) *Request {
	r.onCircularBreakerHalfOpen = f
	return r
}

func (r *Request) OnCircularBreakerClosed(f func()) *Request {
	r.onCircularBreakerClosed = f
	return r
}

func (r *Request) OnCircuitBreakerStateChange(f func(requester *Request, from CBState, to CBState)) *Request {
	r.onCircuitBreakerStateChange = f
	return r
}

func (r *Request) OnRetry(f func()) *Request {
	r.onRetry = f
	return r
}

func (r *Request) OnTimeout(f func()) *Request {
	r.onTimeout = f
	return r
}
