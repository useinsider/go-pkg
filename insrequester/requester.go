package insrequester

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/slok/goresilience"
	"github.com/slok/goresilience/circuitbreaker"
	goresilienceErrors "github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/retry"
	"io"
	"net/http"
	"time"
)

// NewRequester ...
func NewRequester() Requester {
	return &Request{}
}

type CircuitBreakerConfig struct {
	MinimumRequestToOpen         int
	SuccessfulRequiredOnHalfOpen int
	WaitDurationInOpenState      time.Duration
	WrapError                    bool
}

type RetryConfig struct {
	WaitBase time.Duration
	Times    int
}

// Requester represent the package structure, with creating exactly the same interface your own codebase you can
// easily mock the functions inside this package while writing unit tests.
type Requester interface {
	Get(re RequestEntity) (*http.Response, error)
	Post(re RequestEntity) (*http.Response, error)
	Put(re RequestEntity) (*http.Response, error)
	Delete(re RequestEntity) (*http.Response, error)
	WithRetry(config RetryConfig) *Request
	WithCircuitbreaker(config CircuitBreakerConfig) *Request
	WithTimeout(timeout time.Duration) *Request
	WithHeaders(headers Headers) *Request
	Load() *Request
}

type Headers []map[string]interface{}

type Configs struct {
	WrapCBError bool
}

// RequestEntity contains required information for sending http request.
type RequestEntity struct {
	Headers  Headers
	Endpoint string
	Body     []byte
}

type Request struct {
	timeout     time.Duration
	runner      goresilience.Runner
	middlewares []goresilience.Middleware
	configs     Configs
	headers     Headers
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
		r.runner = goresilience.RunnerChain(r.middlewares...)
	}

	runnerErr := r.runner.Run(context.TODO(), func(ctx context.Context) error {
		var req *http.Request

		req, outerErr = http.NewRequest(httpMethod, re.Endpoint, bytes.NewReader(re.Body))
		if outerErr != nil {
			res = nil
			return nil
		}

		req.Close = true
		re.Headers = append(r.headers, re.Headers...) // RequestEntity headers will override Requester level headers.
		re.applyHeadersToRequest(req)

		res, outerErr = (&http.Client{Timeout: r.timeout}).Do(req)
		if outerErr != nil {
			return ErrRetryable
		}

		if res.StatusCode >= 100 && res.StatusCode < 200 ||
			res.StatusCode == 429 ||
			res.StatusCode >= 500 && res.StatusCode <= 599 {
			return ErrRetryable
		}

		return nil
	})

	if runnerErr == goresilienceErrors.ErrCircuitOpen {
		if r.configs.WrapCBError {
			if outerErr != nil {
				return nil, errors.Wrap(ErrCircuitBreakerOpen, outerErr.Error())
			}
			if res != nil && (res.StatusCode >= 100 && res.StatusCode < 200 ||
				res.StatusCode == 429 ||
				res.StatusCode >= 500 && res.StatusCode <= 599) {
				return nil, errors.Wrap(ErrCircuitBreakerOpen, r.getBodyError(*res))
			}
		}
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

func (r *Request) getBodyError(res http.Response) string {
	var err error

	bodyBytes, err := io.ReadAll(res.Body)

	if err != nil {
		return res.Status + " : " + ErrReadingBody.Error()
	}

	return res.Status + " : " + string(bodyBytes)
}

func (r *Request) WithRetry(config RetryConfig) *Request {
	if config.WaitBase == 0 {
		config.WaitBase = 200 * time.Millisecond
	}

	if config.Times == 0 {
		config.Times = 3
	}

	mw := retry.NewMiddleware(retry.Config{
		WaitBase: config.WaitBase,
		Times:    config.Times,
	})
	r.middlewares = append(r.middlewares, mw)

	return r
}
func (r *Request) WithCircuitbreaker(config CircuitBreakerConfig) *Request {
	if config.MinimumRequestToOpen == 0 {
		config.MinimumRequestToOpen = 3
	}

	if config.SuccessfulRequiredOnHalfOpen == 0 {
		config.SuccessfulRequiredOnHalfOpen = 1
	}

	if config.WaitDurationInOpenState == 0 {
		config.WaitDurationInOpenState = 5 * time.Second
	}

	if config.WrapError {
		r.configs.WrapCBError = true
	}

	mw := circuitbreaker.NewMiddleware(circuitbreaker.Config{
		MinimumRequestToOpen:         config.MinimumRequestToOpen,
		SuccessfulRequiredOnHalfOpen: config.SuccessfulRequiredOnHalfOpen,
		WaitDurationInOpenState:      config.WaitDurationInOpenState,
	})
	r.middlewares = append(r.middlewares, mw)

	return r
}

func (r *Request) WithTimeout(timeout time.Duration) *Request {
	if timeout == 0 {
		r.timeout = 30 * time.Second
	} else {
		r.timeout = timeout
	}

	return r
}

func (r *Request) WithHeaders(headers Headers) *Request {
	r.headers = headers
	return r
}

func (r *Request) Load() *Request {
	r.runner = goresilience.RunnerChain(r.middlewares...)
	return r
}
