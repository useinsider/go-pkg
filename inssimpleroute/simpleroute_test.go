package inssimpleroute_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/useinsider/go-pkg/inssimpleroute"
)

type echoCommand struct {
	Name string `json:"name"`
}

type echoResult struct {
	Greeting string `json:"greeting"`
}

func decodeEcho(_ context.Context, r *http.Request) (*echoCommand, error) {
	var cmd echoCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func echoUseCase(_ context.Context, cmd *echoCommand) (echoResult, error) {
	return echoResult{Greeting: "hello " + cmd.Name}, nil
}

type statusErr struct {
	code int
	msg  string
}

func (e statusErr) Error() string { return e.msg }

func (e statusErr) StatusCode() int { return e.code }

func (e statusErr) Headers() http.Header {
	return http.Header{"X-Err": []string{"yes"}}
}

func (e statusErr) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"error": e.msg})
}

type badMarshalErr struct{}

func (badMarshalErr) Error() string                { return "cannot marshal me" }
func (badMarshalErr) MarshalJSON() ([]byte, error) { return nil, errors.New("marshal broken") }

type headeredResult struct {
	Value string `json:"value"`
}

func (headeredResult) Headers() http.Header {
	return http.Header{"X-Result": []string{"a", "b"}}
}

func (headeredResult) StatusCode() int { return http.StatusCreated }

type noContentResult struct{}

func (noContentResult) StatusCode() int { return http.StatusNoContent }

func TestSimpleRoute_ServeHTTP(t *testing.T) {
	t.Run("it_should_serve_decoded_request_through_use_case", func(t *testing.T) {
		srv := inssimpleroute.NewServerWithDefaults(echoUseCase, decodeEcho)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"world"}`))
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
		}
		if body := strings.TrimSpace(rec.Body.String()); body != `{"greeting":"hello world"}` {
			t.Errorf("body = %q, want %q", body, `{"greeting":"hello world"}`)
		}
	})

	t.Run("it_should_encode_decode_error_with_default_error_encoder", func(t *testing.T) {
		srv := inssimpleroute.NewServerWithDefaults(echoUseCase, decodeEcho)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`not-json`))
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", ct)
		}
		if rec.Body.Len() == 0 {
			t.Error("body is empty, want the decode error text")
		}
	})

	t.Run("it_should_encode_use_case_error", func(t *testing.T) {
		failing := func(_ context.Context, _ *echoCommand) (echoResult, error) {
			return echoResult{}, statusErr{code: http.StatusTeapot, msg: "teapot"}
		}
		srv := inssimpleroute.NewServerWithDefaults(failing, decodeEcho)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"}`))
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusTeapot {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q, want JSON content type", ct)
		}
		if got := rec.Header().Get("X-Err"); got != "yes" {
			t.Errorf("X-Err header = %q, want %q", got, "yes")
		}
		if body := strings.TrimSpace(rec.Body.String()); body != `{"error":"teapot"}` {
			t.Errorf("body = %q, want %q", body, `{"error":"teapot"}`)
		}
	})

	t.Run("it_should_encode_encoder_error_through_error_encoder", func(t *testing.T) {
		failEnc := func(_ context.Context, _ http.ResponseWriter, _ echoResult) error {
			return errors.New("encode failed")
		}
		var encoderErr error
		errEnc := func(_ context.Context, err error, w http.ResponseWriter) {
			encoderErr = err
			w.WriteHeader(http.StatusBadGateway)
		}
		srv := inssimpleroute.NewServer(echoUseCase, decodeEcho, failEnc, errEnc)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"}`))
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
		}
		if encoderErr == nil || encoderErr.Error() != "encode failed" {
			t.Errorf("errorEncoder received %v, want encode failed", encoderErr)
		}
	})
}

func TestEncodeJSONResponse(t *testing.T) {
	t.Run("it_should_apply_headers_and_status_code_from_response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := inssimpleroute.EncodeJSONResponse(context.Background(), rec, headeredResult{Value: "v"})
		if err != nil {
			t.Fatalf("EncodeJSONResponse() error = %v", err)
		}

		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
		if got := rec.Header()["X-Result"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("X-Result = %v, want [a b]", got)
		}
		if body := strings.TrimSpace(rec.Body.String()); body != `{"value":"v"}` {
			t.Errorf("body = %q, want %q", body, `{"value":"v"}`)
		}
	})

	t.Run("it_should_skip_body_on_no_content", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := inssimpleroute.EncodeJSONResponse(context.Background(), rec, noContentResult{})
		if err != nil {
			t.Fatalf("EncodeJSONResponse() error = %v", err)
		}

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("body = %q, want empty for 204", rec.Body.String())
		}
	})

	t.Run("it_should_default_to_200_for_plain_response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := inssimpleroute.EncodeJSONResponse(context.Background(), rec, echoResult{Greeting: "hi"}); err != nil {
			t.Fatalf("EncodeJSONResponse() error = %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("it_should_return_encoding_error_for_unmarshalable_response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := inssimpleroute.EncodeJSONResponse(context.Background(), rec, make(chan int))
		if err == nil {
			t.Error("EncodeJSONResponse() error = nil, want json encoding error")
		}
	})
}

func TestDefaultErrorEncoder(t *testing.T) {
	t.Run("it_should_write_plain_text_500_for_plain_error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		inssimpleroute.DefaultErrorEncoder(context.Background(), errors.New("plain failure"), rec)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Errorf("Content-Type = %q, want text/plain", ct)
		}
		body, _ := io.ReadAll(rec.Body)
		if string(body) != "plain failure" {
			t.Errorf("body = %q, want %q", body, "plain failure")
		}
	})

	t.Run("it_should_fall_back_to_plain_text_when_marshaling_fails", func(t *testing.T) {
		rec := httptest.NewRecorder()
		inssimpleroute.DefaultErrorEncoder(context.Background(), badMarshalErr{}, rec)

		if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Errorf("Content-Type = %q, want text/plain fallback", ct)
		}
		if body := rec.Body.String(); body != "cannot marshal me" {
			t.Errorf("body = %q, want the plain error text", body)
		}
	})

	t.Run("it_should_use_json_headers_and_status_from_error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		inssimpleroute.DefaultErrorEncoder(context.Background(), statusErr{code: http.StatusConflict, msg: "dupe"}, rec)

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q, want JSON content type", ct)
		}
		if got := rec.Header().Get("X-Err"); got != "yes" {
			t.Errorf("X-Err = %q, want yes", got)
		}
		if body := rec.Body.String(); body != `{"error":"dupe"}` {
			t.Errorf("body = %q, want %q", body, `{"error":"dupe"}`)
		}
	})
}
