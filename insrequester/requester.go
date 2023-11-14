package insrequester

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/slok/goresilience"
	goresilienceErrors "github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/metrics"
	"net/http"
)

// RequestEntity contains required information for sending http request.
type RequestEntity struct {
	Headers  []map[string]interface{}
	Endpoint string
	Body     []byte
}

type Request struct {
	name                        string
	cfg                         RequesterConfig
	timeout                     int
	runner                      goresilience.Runner
	middlewares                 []goresilience.Middleware
	client                      *http.Client
	onCircularBreakerOpen       func()
	onCircularBreakerHalfOpen   func()
	onCircularBreakerClosed     func()
	onCircuitBreakerStateChange func(requester *Request, from CBState, to CBState)
	cbState                     CBState
	onRetry                     func()
	onTimeout                   func()
	consecutiveSuccessesCount   int
	consecutiveFailuresCount    int
	totalSuccessesCount         int
	totalFailuresCount          int
	totalCount                  int
	cache                       *redis.Client
}

type CBState string

const (
	CBStateOpen     CBState = "open"
	CBStateHalfOpen CBState = "halfopen"
	CBStateClosed   CBState = "closed"
)

// NewRequester ...
func NewRequester(cache *redis.Client) Requester {
	return &Request{
		cache: cache,
	}
}

// Requester represent the package structure, with creating exactly the same interface your own codebase you can
// easily mock the functions inside this package while writing unit tests.
type Requester interface {
	SetName(name string) *Request
	Name() string
	Configure(cfg RequesterConfig) *Request
	Get(re RequestEntity) (*http.Response, error)
	Post(re RequestEntity) (*http.Response, error)
	Put(re RequestEntity) (*http.Response, error)
	Delete(re RequestEntity) (*http.Response, error)
	WithRetry(config RetryConfig) *Request
	WithCircuitbreaker(config CircuitBreakerConfig) *Request
	WithTimeout(timeoutSeconds int) *Request
	OnCircularBreakerOpen(func()) *Request
	OnCircularBreakerHalfOpen(func()) *Request
	OnCircularBreakerClosed(func()) *Request
	OnCircuitBreakerStateChange(func(requester *Request, from CBState, to CBState)) *Request
	OnRetry(func()) *Request
	OnTimeout(func()) *Request
	Load() *Request
}

func (r *Request) SetName(name string) *Request {
	r.name = name
	return r
}

func (r *Request) Name() string {
	return r.name
}

func (r *Request) Configure(cfg RequesterConfig) *Request {
	r.cfg = cfg
	return r
}

// Get sends HTTP get request to the given endpoint and returns *http.Response and an error.
func (r *Request) Get(re RequestEntity) (*http.Response, error) {
	return r.sendRequest(http.MethodGet, re)
}

// Post sends HTTP post request to the given endpoint and returns *http.Response and an error.
func (r *Request) Post(re RequestEntity) (*http.Response, error) {
	return r.sendRequest(http.MethodPost, re)
}

// Put sends HTTP put request to the given endpoint and returns *http.Response and an error.
func (r *Request) Put(re RequestEntity) (*http.Response, error) {
	return r.sendRequest(http.MethodPut, re)
}

// Delete sends HTTP put request to the given endpoint and returns *http.Response and an error.
func (r *Request) Delete(re RequestEntity) (*http.Response, error) {
	return r.sendRequest(http.MethodDelete, re)
}

func (r *Request) sendRequest(httpMethod string, re RequestEntity) (*http.Response, error) {
	var (
		res      *http.Response
		outerErr error
	)

	if r.runner == nil {
		return nil, fmt.Errorf("requester is not loaded")
	}

	runnerErr := r.runner.Run(context.TODO(), func(ctx context.Context) error {
		var req *http.Request

		req, outerErr = http.NewRequest(httpMethod, re.Endpoint, bytes.NewReader(re.Body))
		if outerErr != nil {
			res = nil
			return nil
		}

		req.Close = true
		re.applyHeadersToRequest(req)

		res, outerErr = r.client.Do(req)
		if outerErr != nil {
			return nil
		}

		r.totalCount++

		if res.StatusCode >= 500 {
			r.consecutiveFailuresCount++
			r.totalFailuresCount++
			r.consecutiveSuccessesCount = 0
		}

		if res.StatusCode >= 100 && res.StatusCode < 200 ||
			res.StatusCode == 429 ||
			res.StatusCode >= 500 && res.StatusCode <= 599 {
			return ErrRetryable
		}

		r.consecutiveSuccessesCount++
		r.totalSuccessesCount++
		r.consecutiveFailuresCount = 0

		return nil
	})

	if runnerErr == goresilienceErrors.ErrCircuitOpen {
		return nil, ErrCircuitBreakerOpen
	}

	if runnerErr == goresilienceErrors.ErrTimeout {
		return nil, ErrTimeout
	}

	if outerErr != nil {
		return nil, outerErr
	}

	return res, nil
}

func (r RequestEntity) applyHeadersToRequest(request *http.Request) {
	if request.Body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	for _, header := range r.Headers {
		for key, value := range header {
			if key == "Host" {
				request.Host = fmt.Sprintf("%v", value)
			} else {
				request.Header.Set(key, fmt.Sprintf("%v", value))
			}
		}
	}
}

func (r *Request) Load() *Request {
	metricRecorder := NewRecorder(r)
	m := metrics.NewMiddleware("insrequester", metricRecorder)
	newMiddlewares := []goresilience.Middleware{m}
	r.middlewares = append(newMiddlewares, r.middlewares...)

	var t = http.DefaultTransport.(*http.Transport).Clone()
	if r.cfg.MaxIdleConns > 0 {
		t.MaxIdleConns = r.cfg.MaxIdleConns
	}

	if r.cfg.MaxIdleConnsPerHost > 0 {
		t.MaxIdleConnsPerHost = r.cfg.MaxIdleConnsPerHost
	}

	if r.cfg.MaxConnsPerHost > 0 {
		t.MaxConnsPerHost = r.cfg.MaxConnsPerHost
	}

	r.client = &http.Client{Transport: t}
	r.runner = goresilience.RunnerChain(r.middlewares...)

	return r
}
