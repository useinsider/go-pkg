# Sentry Package


This is a simple sentry wrapper.

## Usage in Apps
```go
package main

import (
	"errors"
	"github.com/useinsider/go-pkg/inssentry"
	"time"
)

func main() {
	sentrySettings := inssentry.Settings{
		SentryDsn:        "sentry_dsn",
		AttachStacktrace: true,
		FlushInterval:    2 * time.Second,
		IsProduction:     false,
	}
	err := inssentry.Init(sentrySettings)
	if err != nil {
		panic(err)
	}

	defer inssentry.Flush()

	inssentry.Error(errors.New("test_error"))
}
```
