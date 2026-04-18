package insrequester

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	pkgerrors "github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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
	timeout    time.Duration
	httpClient *http.Client
	executor   failsafe.Executor[*http.Response]
	policies   []failsafe.Policy[*http.Response]
	headers    Headers
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
		outerErr    error
		attempt     int
		lastErrBody string
	)

	if r.executor == nil {
		r.executor = failsafe.NewExecutor[*http.Response](r.policies...)
	}

	res, runnerErr := r.executor.WithContext(ctx).GetWithExecution(func(exec failsafe.Execution[*http.Response]) (*http.Response, error) {
		attempt++

		req, err := http.NewRequestWithContext(exec.Context(), httpMethod, re.Endpoint, bytes.NewReader(re.Body))
		if err != nil {
			outerErr = err
			return nil, nil
		}

		req.Close = true
		otel.GetTextMapPropagator().Inject(exec.Context(), propagation.HeaderCarrier(req.Header))
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

		response, doErr := client.Do(req)
		if doErr != nil {
			outerErr = doErr
			return nil, ErrRetryable
		}

		if response.StatusCode >= 100 && response.StatusCode < 200 ||
			response.StatusCode == 429 ||
			response.StatusCode >= 500 && response.StatusCode <= 599 {
			const maxErrBodySize = 4096
			limitedReader := io.LimitReader(response.Body, maxErrBodySize+1)
			bodyBytes, _ := io.ReadAll(limitedReader)
			response.Body.Close()
			truncated := len(bodyBytes) > maxErrBodySize
			if truncated {
				bodyBytes = bodyBytes[:maxErrBodySize]
			}
			if len(bodyBytes) > 0 {
				msg := fmt.Sprintf("%s : %s", response.Status, string(bodyBytes))
				if truncated {
					msg += " [truncated]"
				}
				lastErrBody = msg
			} else {
				lastErrBody = response.Status
			}
			return response, ErrRetryable
		}

		return response, nil
	})

	resendCount := 0
	if attempt > 0 {
		resendCount = attempt - 1
	}
	span.SetAttributes(attribute.Int("http.resend_count", resendCount))

	if errors.Is(runnerErr, circuitbreaker.ErrOpen) {
		span.SetStatus(codes.Error, "circuit breaker open")
		if outerErr != nil {
			return nil, pkgerrors.Wrap(ErrCircuitBreakerOpen, outerErr.Error())
		}
		if lastErrBody != "" {
			return nil, pkgerrors.Wrap(ErrCircuitBreakerOpen, lastErrBody)
		}
		return nil, ErrCircuitBreakerOpen
	}

	if outerErr != nil {
		span.RecordError(outerErr)
		span.SetStatus(codes.Error, outerErr.Error())
		return nil, outerErr
	}

	if runnerErr != nil {
		span.SetStatus(codes.Error, "retries exhausted")
		if lastErrBody != "" {
			return nil, pkgerrors.Wrap(ErrRetriesExhausted, lastErrBody)
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

	policy := retrypolicy.Builder[*http.Response]().
		WithMaxRetries(config.Times).
		WithDelay(config.WaitBase).
		HandleIf(func(_ *http.Response, err error) bool {
			return err != nil
		}).
		Build()

	r.policies = append(r.policies, policy)

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

	policy := circuitbreaker.Builder[*http.Response]().
		WithFailureThreshold(uint(config.MinimumRequestToOpen)).
		WithSuccessThreshold(uint(config.SuccessfulRequiredOnHalfOpen)).
		WithDelay(config.WaitDurationInOpenState).
		HandleIf(func(_ *http.Response, err error) bool {
			return err != nil
		}).
		Build()

	r.policies = append(r.policies, policy)

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
	r.executor = failsafe.NewExecutor[*http.Response](r.policies...)
	return r
}
