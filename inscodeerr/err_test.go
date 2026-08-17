package inscodeerr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/useinsider/go-pkg/inscodeerr"
)

func TestNewCodeErr(t *testing.T) {
	t.Run("it_should_set_all_fields", func(t *testing.T) {
		wrapped := errors.New("boom")
		c := inscodeerr.NewCodeErr(http.StatusBadRequest, wrapped, "bad input")

		if c.Code != http.StatusBadRequest {
			t.Errorf("Code = %d, want %d", c.Code, http.StatusBadRequest)
		}
		if c.Err != wrapped {
			t.Errorf("Err = %v, want %v", c.Err, wrapped)
		}
		if c.Message != "bad input" {
			t.Errorf("Message = %q, want %q", c.Message, "bad input")
		}
	})
}

func TestCodeErr_Error(t *testing.T) {
	tests := []struct {
		name string
		in   inscodeerr.CodeErr
		want string
	}{
		{
			name: "it_should_include_status_text_for_known_code",
			in:   inscodeerr.CodeErr{Code: http.StatusNotFound},
			want: "HTTP 404: Not Found",
		},
		{
			name: "it_should_omit_status_text_for_unknown_code",
			in:   inscodeerr.CodeErr{Code: 599},
			want: "HTTP 599",
		},
		{
			name: "it_should_append_wrapped_error",
			in:   inscodeerr.CodeErr{Code: http.StatusInternalServerError, Err: errors.New("boom")},
			want: "HTTP 500: Internal Server Error: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCodeErr_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   inscodeerr.CodeErr
		want string
	}{
		{
			name: "it_should_report_status_true_for_2xx",
			in:   inscodeerr.CodeErr{Code: http.StatusOK, Message: "done"},
			want: `{"status":true,"message":"done"}`,
		},
		{
			name: "it_should_report_status_false_below_200",
			in:   inscodeerr.CodeErr{Code: http.StatusContinue, Message: "hold"},
			want: `{"status":false,"message":"hold"}`,
		},
		{
			name: "it_should_report_status_false_for_4xx",
			in:   inscodeerr.CodeErr{Code: http.StatusForbidden, Message: "denied"},
			want: `{"status":false,"message":"denied"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestCodeErr_StatusCode(t *testing.T) {
	t.Run("it_should_return_the_code", func(t *testing.T) {
		c := inscodeerr.CodeErr{Code: http.StatusTeapot}
		if got := c.StatusCode(); got != http.StatusTeapot {
			t.Errorf("StatusCode() = %d, want %d", got, http.StatusTeapot)
		}
	})
}

func TestCodeErr_Headers(t *testing.T) {
	t.Run("it_should_return_empty_non_nil_headers", func(t *testing.T) {
		c := inscodeerr.CodeErr{}
		h := c.Headers()
		if h == nil {
			t.Fatal("Headers() = nil, want empty http.Header")
		}
		if len(h) != 0 {
			t.Errorf("Headers() has %d entries, want 0", len(h))
		}
	})
}

func TestGetStatusCode(t *testing.T) {
	t.Run("it_should_return_code_for_CodeErr_value", func(t *testing.T) {
		err := inscodeerr.NewCodeErr(http.StatusConflict, nil, "")
		if got := inscodeerr.GetStatusCode(err); got != http.StatusConflict {
			t.Errorf("GetStatusCode() = %d, want %d", got, http.StatusConflict)
		}
	})

	t.Run("it_should_fall_back_to_500_for_plain_error", func(t *testing.T) {
		if got := inscodeerr.GetStatusCode(errors.New("plain")); got != http.StatusInternalServerError {
			t.Errorf("GetStatusCode() = %d, want %d", got, http.StatusInternalServerError)
		}
	})

	t.Run("it_should_fall_back_to_500_for_pointer_to_CodeErr", func(t *testing.T) {
		// PA-39500: pins current behavior — GetStatusCode uses a plain type
		// assertion on the CodeErr value type, so *CodeErr (and wrapped
		// CodeErr via fmt.Errorf %w) fall back to 500 instead of the code.
		err := &inscodeerr.CodeErr{Code: http.StatusConflict}
		if got := inscodeerr.GetStatusCode(err); got != http.StatusInternalServerError {
			t.Errorf("GetStatusCode(*CodeErr) = %d, want %d", got, http.StatusInternalServerError)
		}
	})
}
