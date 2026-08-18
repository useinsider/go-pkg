package inssentry

import (
	"errors"
	"testing"
	"time"
)


func initFor(t *testing.T, production bool) {
	t.Helper()
	err := Init(Settings{
		SentryDsn:        "",
		AttachStacktrace: true,
		FlushInterval:    10 * time.Millisecond,
		IsProduction:     production,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
}

func TestInit(t *testing.T) {
	t.Run("it_should_cache_settings_and_open_disabled_client", func(t *testing.T) {
		settings := Settings{
			SentryDsn:        "",
			AttachStacktrace: true,
			FlushInterval:    5 * time.Millisecond,
			IsProduction:     true,
		}

		if err := Init(settings); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
		if cachedSettings != settings {
			t.Errorf("cachedSettings = %+v, want %+v", cachedSettings, settings)
		}
	})

	t.Run("it_should_return_error_for_malformed_dsn", func(t *testing.T) {
		err := Init(Settings{SentryDsn: "://not-a-dsn"})
		if err == nil {
			t.Error("Init() error = nil, want DSN parse error")
		}
	})
}

func TestFlush(t *testing.T) {
	t.Run("it_should_flush_without_panicking", func(t *testing.T) {
		initFor(t, false)
		Flush()
	})
}

func TestError(t *testing.T) {
	t.Run("it_should_log_locally_when_not_production", func(t *testing.T) {
		initFor(t, false)
		Error(errors.New("non-production error"))
	})

	t.Run("it_should_capture_when_production", func(t *testing.T) {
		initFor(t, true)
		Error(errors.New("production error"))
	})
}

func TestErrorWithAdditionalData(t *testing.T) {
	t.Run("it_should_log_locally_when_not_production", func(t *testing.T) {
		initFor(t, false)
		ErrorWithAdditionalData(errors.New("non-production error"), "key", "value")
	})

	t.Run("it_should_capture_with_scope_when_production", func(t *testing.T) {
		initFor(t, true)
		ErrorWithAdditionalData(errors.New("production error"), "key", map[string]int{"n": 1})
	})
}

func TestFatal(t *testing.T) {
	t.Run("it_should_panic_when_not_production", func(t *testing.T) {
		initFor(t, false)

		sentinel := errors.New("fatal in dev")
		defer func() {
			if r := recover(); r != sentinel {
				t.Errorf("recover() = %v, want the original error", r)
			}
		}()

		Fatal(sentinel)
	})

	t.Run("it_should_capture_then_panic_when_production", func(t *testing.T) {
		initFor(t, true)

		sentinel := errors.New("fatal in prod")
		defer func() {
			if r := recover(); r != sentinel {
				t.Errorf("recover() = %v, want the original error", r)
			}
		}()

		Fatal(sentinel)
	})
}
