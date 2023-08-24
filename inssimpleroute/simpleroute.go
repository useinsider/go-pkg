package inssimpleroute

import (
	"context"
	"encoding/json"
	"net/http"
)

type UseCase[UseCaseCommand, UseCaseResult any] func(ctx context.Context, cmd *UseCaseCommand) (rs UseCaseResult, err error)

// SimpleRoute wraps an endpoint and implements http.HttpHandler.
type SimpleRoute[UseCaseCommand, UseCaseResult any] struct {
	e            UseCase[UseCaseCommand, UseCaseResult]
	dec          DecodeRequestFunc[UseCaseCommand]
	enc          EncodeResponseFunc[UseCaseResult]
	errorEncoder ErrorEncoder
}

// NewServerWithDefaults constructs a new server, which implements http.HttpHandler and wraps
// the provided endpoint.
func NewServerWithDefaults[UseCaseCommand, UseCaseResult any](
	e UseCase[UseCaseCommand, UseCaseResult],
	dec DecodeRequestFunc[UseCaseCommand],
) *SimpleRoute[UseCaseCommand, UseCaseResult] {
	return NewServer(e, dec, EncodeJSONResponse[UseCaseResult], DefaultErrorEncoder)
}

// NewServer constructs a new server, which implements http.HttpHandler and wraps
// the provided endpoint.
func NewServer[UseCaseCommand, UseCaseResult any](
	e UseCase[UseCaseCommand, UseCaseResult],
	dec DecodeRequestFunc[UseCaseCommand],
	enc EncodeResponseFunc[UseCaseResult],
	ee ErrorEncoder,
) *SimpleRoute[UseCaseCommand, UseCaseResult] {
	s := &SimpleRoute[UseCaseCommand, UseCaseResult]{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: ee,
	}

	return s
}

// ServeHTTP implements http.HttpHandler.
func (sr *SimpleRoute[UseCaseCommand, UseCaseResult]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	request, err := sr.dec(ctx, r)
	if err != nil {
		sr.errorEncoder(ctx, err, w)
		return
	}

	response, err := sr.e(ctx, request)
	if err != nil {
		sr.errorEncoder(ctx, err, w)
		return
	}

	if err := sr.enc(ctx, w, response); err != nil {
		sr.errorEncoder(ctx, err, w)
		return
	}
}

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error, w http.ResponseWriter)

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object to the ResponseWriter. Many JSON-over-HTTP services can use it as
// a sensible default. If the response implements Headerer, the provided headers
// will be applied to the response. If the response implements StatusCoder, the
// provided StatusCode will be used instead of 200.
func EncodeJSONResponse[UseCaseResult any](_ context.Context, w http.ResponseWriter, response UseCaseResult) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if headerer, ok := any(response).(Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}

	code := http.StatusOK
	if sc, ok := any(response).(StatusCoder); ok {
		code = sc.StatusCode()
	}

	w.WriteHeader(code)

	if code == http.StatusNoContent {
		return nil
	}

	return json.NewEncoder(w).Encode(response)
}

// DefaultErrorEncoder writes the error to the ResponseWriter, by default a
// content type of text/plain, a body of the plain text of the error, and a
// status code of 500. If the error implements Headerer, the provided headers
// will be applied to the response. If the error implements json.Marshaler, and
// the marshaling succeeds, a content type of application/json and the JSON
// encoded form of the error will be used. If the error implements StatusCoder,
// the provided StatusCode will be used instead of 500.
func DefaultErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	contentType, body := "text/plain; charset=utf-8", []byte(err.Error())

	if marshaler, ok := err.(json.Marshaler); ok {
		if jsonBody, marshalErr := marshaler.MarshalJSON(); marshalErr == nil {
			contentType, body = "application/json; charset=utf-8", jsonBody
		}
	}

	w.Header().Set("Content-Type", contentType)

	if headerer, ok := err.(Headerer); ok {
		for k, values := range headerer.Headers() {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}

	code := http.StatusInternalServerError
	if sc, ok := err.(StatusCoder); ok {
		code = sc.StatusCode()
	}

	w.WriteHeader(code)
	_, _ = w.Write(body)
}

// StatusCoder is checked by DefaultErrorEncoder. If an error value implements
// StatusCoder, the StatusCode will be used when encoding the error. By default,
// StatusInternalServerError (500) is used.
type StatusCoder interface {
	StatusCode() int
}

// Headerer is checked by DefaultErrorEncoder. If an error value implements
// Headerer, the provided headers will be applied to the response writer, after
// the Content-Type is set.
type Headerer interface {
	Headers() http.Header
}
