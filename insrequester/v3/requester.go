package insrequester

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("insrequester")

func NewRequester() Requester {
	return &Request{}
}

type CircuitBreakerConfig struct {
	// MinimumRequestToOpen is the number of consecutive failures that will trip the
	// breaker. Used only when rate-based fields below are zero-valued.
	MinimumRequestToOpen int

	// FailureRateThreshold (percent, 1-100), FailureExecutionThreshold (minimum
	// samples in the window), and FailureThresholdingPeriod together configure a
	// Hystrix-style rate-based breaker. When FailureRateThreshold is non-zero,
	// MinimumRequestToOpen is ignored.
	FailureRateThreshold      uint
	FailureExecutionThreshold uint
	FailureThresholdingPeriod time.Duration

	SuccessfulRequiredOnHalfOpen int
	WaitDurationInOpenState      time.Duration
}

type RetryConfig struct {
	// WaitBase is the base delay between retries. When WaitMax is zero the delay is
	// fixed; otherwise delay grows exponentially up to WaitMax.
	WaitBase time.Duration
	// WaitMax caps exponential backoff. Zero disables backoff (fixed WaitBase).
	WaitMax time.Duration
	// JitterFactor randomizes each delay by +/- (delay * factor). Valid range: 0.0-1.0.
	JitterFactor float32
	Times        int
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
	initOnce   sync.Once
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

// Delete sends HTTP delete request to the given endpoint and returns *http.Response and an error.
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
		outerErr       error
		attempt        int
		lastErrBody    string
		lastStatusCode int
	)

	r.initOnce.Do(func() {
		if r.executor == nil {
			r.executor = failsafe.NewExecutor[*http.Response](r.policies...)
		}
	})

	res, runnerErr := r.executor.WithContext(ctx).GetWithExecution(func(exec failsafe.Execution[*http.Response]) (*http.Response, error) {
		attempt++

		req, err := http.NewRequestWithContext(exec.Context(), httpMethod, re.Endpoint, bytes.NewReader(re.Body))
		outerErr = err
		if err != nil {
			return nil, nil
		}

		req.Close = true
		otel.GetTextMapPropagator().Inject(exec.Context(), propagation.HeaderCarrier(req.Header))
		combinedHeaders := make(Headers, 0, len(r.headers)+len(re.Headers))
		combinedHeaders = append(combinedHeaders, r.headers...)
		combinedHeaders = append(combinedHeaders, re.Headers...) // RequestEntity headers will override Requester level headers.
		RequestEntity{Headers: combinedHeaders}.applyHeadersToRequest(req)

		client := r.httpClient
		if client == nil {
			client = &http.Client{Timeout: r.timeout}
		} else if r.timeout > 0 {
			cp := *client
			cp.Timeout = r.timeout
			client = &cp
		}

		response, doErr := client.Do(req)
		outerErr = doErr
		if doErr != nil {
			if response != nil && response.Body != nil {
				response.Body.Close()
			}
			if ctxErr := exec.Context().Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, ErrRetryable
		}

		lastStatusCode = response.StatusCode

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
	if lastStatusCode > 0 {
		span.SetAttributes(attribute.Int("http.response.status_code", lastStatusCode))
	}

	if errors.Is(runnerErr, circuitbreaker.ErrOpen) {
		span.SetStatus(codes.Error, "circuit breaker open")
		if outerErr != nil {
			return nil, fmt.Errorf("%s: %w", outerErr.Error(), ErrCircuitBreakerOpen)
		}
		if lastErrBody != "" {
			return nil, fmt.Errorf("%s: %w", lastErrBody, ErrCircuitBreakerOpen)
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
			return nil, fmt.Errorf("%s: %w", lastErrBody, ErrRetriesExhausted)
		}
		return nil, ErrRetriesExhausted
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

	builder := retrypolicy.Builder[*http.Response]().
		WithMaxRetries(config.Times).
		HandleIf(func(_ *http.Response, err error) bool {
			return err != nil
		}).
		AbortOnErrors(circuitbreaker.ErrOpen, context.Canceled, context.DeadlineExceeded)

	if config.WaitMax > 0 {
		builder = builder.WithBackoff(config.WaitBase, config.WaitMax)
	} else {
		builder = builder.WithDelay(config.WaitBase)
	}

	if config.JitterFactor > 0 {
		jitter := config.JitterFactor
		if jitter > 1 {
			jitter = 1
		}
		builder = builder.WithJitterFactor(jitter)
	}

	policy := builder.Build()

	r.policies = append(r.policies, policy)

	return r
}

func (r *Request) WithCircuitbreaker(config CircuitBreakerConfig) *Request {
	if config.SuccessfulRequiredOnHalfOpen == 0 {
		config.SuccessfulRequiredOnHalfOpen = 1
	}

	if config.WaitDurationInOpenState == 0 {
		config.WaitDurationInOpenState = 5 * time.Second
	}

	successThreshold := config.SuccessfulRequiredOnHalfOpen
	if successThreshold < 0 {
		successThreshold = 0
	}
	builder := circuitbreaker.Builder[*http.Response]().
		WithSuccessThreshold(uint(successThreshold)).
		WithDelay(config.WaitDurationInOpenState).
		HandleIf(func(_ *http.Response, err error) bool {
			return err != nil
		})

	if config.FailureRateThreshold > 0 {
		rate := config.FailureRateThreshold
		if rate > 100 {
			rate = 100
		}
		if config.FailureExecutionThreshold == 0 {
			config.FailureExecutionThreshold = 20
		}
		if config.FailureThresholdingPeriod == 0 {
			config.FailureThresholdingPeriod = 10 * time.Second
		}
		builder = builder.WithFailureRateThreshold(
			rate,
			config.FailureExecutionThreshold,
			config.FailureThresholdingPeriod,
		)
	} else {
		if config.MinimumRequestToOpen == 0 {
			config.MinimumRequestToOpen = 3
		}
		if config.MinimumRequestToOpen < 0 {
			config.MinimumRequestToOpen = 0
		}
		builder = builder.WithFailureThreshold(uint(config.MinimumRequestToOpen))
	}

	r.policies = append(r.policies, builder.Build())

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
