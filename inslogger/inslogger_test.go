package inslogger

import (
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newObservedLogger() (*AppLogger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core, zap.WithFatalHook(zapcore.WriteThenPanic))
	return &AppLogger{Logger: logger, Sugar: logger.Sugar(), Level: Debug}, logs
}

func TestLogLevel_toZapLevel(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  zapcore.Level
	}{
		{"it_should_map_debug", Debug, zapcore.DebugLevel},
		{"it_should_map_info", Info, zapcore.InfoLevel},
		{"it_should_map_warn", Warn, zapcore.WarnLevel},
		{"it_should_map_error", Error, zapcore.ErrorLevel},
		{"it_should_map_fatal", Fatal, zapcore.FatalLevel},
		{"it_should_default_unknown_levels_to_info", LogLevel("NOPE"), zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.toZapLevel().Level(); got != tt.want {
				t.Errorf("toZapLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	levels := []struct {
		name  string
		level LogLevel
	}{
		{"it_should_build_debug_logger", Debug},
		{"it_should_build_info_logger", Info},
		{"it_should_build_warn_logger", Warn},
		{"it_should_build_error_logger", Error},
		{"it_should_build_fatal_logger", Fatal},
		{"it_should_build_default_logger_for_unknown_level", LogLevel("NOPE")},
	}

	for _, tt := range levels {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLogger(tt.level)
			al, ok := l.(*AppLogger)
			if !ok {
				t.Fatalf("NewLogger() returned %T, want *AppLogger", l)
			}
			if al.Logger == nil || al.Sugar == nil {
				t.Error("NewLogger() left Logger/Sugar nil")
			}
			if al.Level != tt.level {
				t.Errorf("Level = %q, want %q", al.Level, tt.level)
			}
		})
	}
}

func TestNewNopLogger(t *testing.T) {
	t.Run("it_should_build_a_safe_noop_logger", func(t *testing.T) {
		l := NewNopLogger()
		if l == nil {
			t.Fatal("NewNopLogger() = nil")
		}
		l.Log("x")
		l.Logf("%s", "x")
		l.Warn("x")
		l.Warnf("%s", "x")
		l.Error(errors.New("x"))
		l.Errorf("%s", "x")
		l.Debug("x")
		l.Debugf("%s", "x")
		l.LogMultiple([]error{errors.New("a")})
	})
}

func TestAppLogger_LoggingMethods(t *testing.T) {
	tests := []struct {
		name      string
		log       func(al *AppLogger)
		wantLevel zapcore.Level
		wantMsg   string
		wantCount int
	}{
		{
			name:      "it_should_log_info_via_Log",
			log:       func(al *AppLogger) { al.Log("hello") },
			wantLevel: zapcore.InfoLevel,
			wantMsg:   "hello",
			wantCount: 1,
		},
		{
			name:      "it_should_log_formatted_info_via_Logf",
			log:       func(al *AppLogger) { al.Logf("hello %d", 42) },
			wantLevel: zapcore.InfoLevel,
			wantMsg:   "hello 42",
			wantCount: 1,
		},
		{
			name:      "it_should_log_warning_via_Warn",
			log:       func(al *AppLogger) { al.Warn("careful") },
			wantLevel: zapcore.WarnLevel,
			wantMsg:   "careful",
			wantCount: 1,
		},
		{
			name:      "it_should_log_formatted_warning_via_Warnf",
			log:       func(al *AppLogger) { al.Warnf("careful %s", "now") },
			wantLevel: zapcore.WarnLevel,
			wantMsg:   "careful now",
			wantCount: 1,
		},
		{
			name:      "it_should_log_error_via_Error",
			log:       func(al *AppLogger) { al.Error(errors.New("kaboom")) },
			wantLevel: zapcore.ErrorLevel,
			wantMsg:   "kaboom+",
			wantCount: 1,
		},
		{
			name:      "it_should_log_formatted_error_via_Errorf",
			log:       func(al *AppLogger) { al.Errorf("kaboom %d", 7) },
			wantLevel: zapcore.ErrorLevel,
			wantMsg:   "kaboom 7",
			wantCount: 1,
		},
		{
			name:      "it_should_log_debug_via_Debug",
			log:       func(al *AppLogger) { al.Debug("verbose") },
			wantLevel: zapcore.DebugLevel,
			wantMsg:   "verbose",
			wantCount: 1,
		},
		{
			name:      "it_should_log_formatted_debug_via_Debugf",
			log:       func(al *AppLogger) { al.Debugf("verbose %v", true) },
			wantLevel: zapcore.DebugLevel,
			wantMsg:   "verbose true",
			wantCount: 1,
		},
		{
			name: "it_should_log_each_error_via_LogMultiple",
			log: func(al *AppLogger) {
				al.LogMultiple([]error{errors.New("one"), errors.New("two")})
			},
			wantLevel: zapcore.InfoLevel,
			wantMsg:   "one+",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			al, logs := newObservedLogger()
			tt.log(al)

			entries := logs.All()
			if len(entries) != tt.wantCount {
				t.Fatalf("logged %d entries, want %d", len(entries), tt.wantCount)
			}
			if entries[0].Level != tt.wantLevel {
				t.Errorf("level = %v, want %v", entries[0].Level, tt.wantLevel)
			}
			if entries[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", entries[0].Message, tt.wantMsg)
			}
		})
	}
}

func TestAppLogger_Fatal(t *testing.T) {
	t.Run("it_should_write_fatal_entry_via_Fatal", func(t *testing.T) {
		al, logs := newObservedLogger()

		defer func() {
			if recover() == nil {
				t.Fatal("Fatal() did not go through the fatal hook")
			}
			entries := logs.All()
			if len(entries) != 1 || entries[0].Level != zapcore.FatalLevel {
				t.Errorf("entries = %+v, want one fatal entry", entries)
			}
		}()

		al.Fatal(errors.New("fatal failure"))
	})

	t.Run("it_should_write_fatal_entry_via_Fatalf", func(t *testing.T) {
		al, logs := newObservedLogger()

		defer func() {
			if recover() == nil {
				t.Fatal("Fatalf() did not go through the fatal hook")
			}
			entries := logs.All()
			if len(entries) != 1 || entries[0].Level != zapcore.FatalLevel {
				t.Errorf("entries = %+v, want one fatal entry", entries)
			}
		}()

		al.Fatalf("fatal %s", "failure")
	})
}

func TestAppLogger_SetLevel(t *testing.T) {
	t.Run("it_should_not_change_the_active_level", func(t *testing.T) {
		al := NewLogger(Info).(*AppLogger)

		if al.Logger.Core().Enabled(zapcore.DebugLevel) {
			t.Fatal("precondition failed: Info logger already accepts debug")
		}

		al.SetLevel(Debug)

		if al.Logger.Core().Enabled(zapcore.DebugLevel) {
			t.Error("SetLevel(Debug) changed the active level; the pinned no-op behavior has been fixed — update this test and the bug registry")
		}
	})
}
