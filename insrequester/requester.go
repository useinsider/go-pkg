package insrequester

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"github.com/slok/goresilience"
	"github.com/slok/goresilience/circuitbreaker"
	goresilienceErrors "github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/retry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
	"time"
)

var tracer = otel.Tracer("insrequester")

// NewRequester ...
func NewRequester() Requester {
	return &Request{}
}

type CircuitBreakerConfig struct {
	MinimumRequestToOpen         int
	SuccessfulRequiredOnHalfOpen int
	WaitDurationInOpenState      time.Duration
}

type RetryConfig struct {
	WaitBase time.Duration
	Times    int
}

// Requester represent the package structure, with creating exactly the same interface your own codebase you can
// easily mock the functions inside this package while writing unit tests.
type Requester interface {
	Get(ctx context.Context, re RequestEntity) (*http.Response, error)
	Post(ctx context.Context, re RequestEntity) (*http.Response, error)
	Put(ctx context.Context, re RequestEntity) (*http.Response, error)
	Delete(ctx context.Context, re RequestEntity) (*http.Response, error)
	WithRetry(config RetryConfig) *Request
	WithCircuitbreaker(config CircuitBreakerConfig) *Request
	WithTimeout(timeout time.Duration) *Request
	WithHTTPClient(client *http.Client) *Request
	WithHeaders(headers Headers) *Request
	Load() *Request
}

type Headers []map[string]interface{}

// RequestEntity contains required information for sending http request.
type RequestEntity struct {
	Headers  Headers
	Endpoint string
	Body     []byte
}

type Request struct {
	timeout     time.Duration
	httpClient  *http.Client
	runner      goresilience.Runner
	middlewares []goresilience.Middleware
	headers     Headers
}

// Get sends HTTP get request to the given endpoint and returns *http.Response and an error.
func (r *Request) Get(ctx context.Context, re RequestEntity) (*http.Response, error) {
	return r.sendRequest(ctx, http.MethodGet, re)
}

// Post sends HTTP post request to the given endpoint and returns *http.Response and an error.
func (r *Request) Post(ctx context.Context, re RequestEntity) (*http.Response, error) {
	return r.sendRequest(ctx, http.MethodPost, re)
}

// Put sends HTTP put request to the given endpoint and returns *http.Response and an error.
func (r *Request) Put(ctx context.Context, re RequestEntity) (*http.Response, error) {
	return r.sendRequest(ctx, http.MethodPut, re)
}

// Delete sends HTTP put request to the given endpoint and returns *http.Response and an error.
func (r *Request) Delete(ctx context.Context, re RequestEntity) (*http.Response, error) {
	return r.sendRequest(ctx, http.MethodDelete, re)
}

func (r *Request) sendRequest(ctx context.Context, httpMethod string, re RequestEntity) (*http.Response, error) {
	spanName := httpMethod
	if parsed, err := url.Parse(re.Endpoint); err == nil {
		spanName = httpMethod + " " + parsed.Host + parsed.Path
	}

	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(
		attribute.String("http.request.method", httpMethod),
		attribute.String("url.full", re.Endpoint),
	))
	defer span.End()

	var (
		res          *http.Response
		outerErr     error
		attempt      int
		lastErrBody  string
	)

	if r.runner == nil {
		r.runner = goresilience.RunnerChain(r.middlewares...)
	}

	runnerErr := r.runner.Run(ctx, func(attemptCtx context.Context) error {
		attempt++
		var req *http.Request

		req, outerErr = http.NewRequestWithContext(attemptCtx, httpMethod, re.Endpoint, bytes.NewReader(re.Body))
		if outerErr != nil {
			res = nil
			return nil
		}

		req.Close = true
		otel.GetTextMapPropagator().Inject(attemptCtx, propagation.HeaderCarrier(req.Header))
		re.Headers = append(r.headers, re.Headers...) // RequestEntity headers will override Requester level headers.
		re.applyHeadersToRequest(req)

		client := r.httpClient
		if client == nil {
			client = &http.Client{Timeout: r.timeout}
		} else if r.timeout > 0 {
			cp := *client
			cp.Timeout = r.timeout
			client = &cp
		}
		res, outerErr = client.Do(req)
		if outerErr != nil {
			return ErrRetryable
		}

		if res.StatusCode >= 100 && res.StatusCode < 200 ||
			res.StatusCode == 429 ||
			res.StatusCode >= 500 && res.StatusCode <= 599 {
			const maxErrBodySize = 4096
			limitedReader := io.LimitReader(res.Body, maxErrBodySize+1)
			bodyBytes, _ := io.ReadAll(limitedReader)
			res.Body.Close()
			truncated := len(bodyBytes) > maxErrBodySize
			if truncated {
				bodyBytes = bodyBytes[:maxErrBodySize]
			}
			if len(bodyBytes) > 0 {
				msg := fmt.Sprintf("%s : %s", res.Status, string(bodyBytes))
				if truncated {
					msg += " [truncated]"
				}
				lastErrBody = msg
			} else {
				lastErrBody = res.Status
			}
			return ErrRetryable
		}

		return nil
	})

	resendCount := 0
	if attempt > 0 {
		resendCount = attempt - 1
	}
	span.SetAttributes(attribute.Int("http.resend_count", resendCount))

	if runnerErr == goresilienceErrors.ErrCircuitOpen {
		span.SetStatus(codes.Error, "circuit breaker open")
		if outerErr != nil {
			return nil, errors.Wrap(ErrCircuitBreakerOpen, outerErr.Error())
		}
		if lastErrBody != "" {
			return nil, errors.Wrap(ErrCircuitBreakerOpen, lastErrBody)
		}
		return nil, ErrCircuitBreakerOpen
	}

	if runnerErr == goresilienceErrors.ErrTimeout {
		span.SetStatus(codes.Error, "timeout")
		return nil, ErrTimeout
	}

	if outerErr != nil {
		span.RecordError(outerErr)
		span.SetStatus(codes.Error, outerErr.Error())
		return nil, outerErr
	}

	if runnerErr != nil {
		span.SetStatus(codes.Error, "retries exhausted")
		if lastErrBody != "" {
			return nil, errors.Wrap(ErrRetriesExhausted, lastErrBody)
		}
		return nil, ErrRetriesExhausted
	}

	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
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

	mw := circuitbreaker.NewMiddleware(circuitbreaker.Config{
		MinimumRequestToOpen:         config.MinimumRequestToOpen,
		SuccessfulRequiredOnHalfOpen: config.SuccessfulRequiredOnHalfOpen,
		WaitDurationInOpenState:      config.WaitDurationInOpenState,
	})
	r.middlewares = append(r.middlewares, mw)

	return r
}

func (r *Request) WithHTTPClient(client *http.Client) *Request {
	r.httpClient = client
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
