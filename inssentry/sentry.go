package inssentry

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"time"
)

// cachedSettings is a global cache for sentry settings.
var cachedSettings Settings

// Settings is a struct for storing sentry options.
type Settings struct {
	SentryDsn        string
	AttachStacktrace bool
	FlushInterval    time.Duration
	IsProduction     bool
}

// Init caches the settings and opens the sentry client.
func Init(settings Settings) error {
	cachedSettings = settings

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cachedSettings.SentryDsn,
		AttachStacktrace: cachedSettings.AttachStacktrace,
	})

	return err
}

// Flush sends the collected sentry messages to sentry.
func Flush() {
	sentry.Flush(cachedSettings.FlushInterval)
}

// Error sends the error to sentry.
func Error(err error) {
	if !cachedSettings.IsProduction {
		fmt.Println(err)

		return
	}

	sentry.CaptureException(err)
}

// ErrorWithAdditionalData sends the error to sentry with additional data.
func ErrorWithAdditionalData(err error, key string, value interface{}) {
	if !cachedSettings.IsProduction {
		fmt.Println(err)

		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetExtra(key, value)

		sentry.CaptureException(err)
	})
}

// Fatal sends the error to sentry and exit from the program.
func Fatal(err error) {
	if !cachedSettings.IsProduction {
		panic(err)

		return
	}

	sentry.CaptureException(err)
	panic(err)
}
