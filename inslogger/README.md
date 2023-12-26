# Inslogger

Inslogger is a logging package designed to provide flexible and comprehensive logging capabilities in Go applications.

## Features

- **Multi-Level Logging:** Supports logging at various levels including Debug, Info, Warn, Error, and Fatal.
- **Error Handling:** Provides methods to log errors with different formatting options.

## Installation

```bash
go get github.com/yourusername/inslogger
```

## Usage

```go
import "github.com/yourusername/inslogger"

// Initialize a logger
logger := inslogger.NewLogger(inslogger.Debug)

// Log an error
err := someFunction()
if err != nil {
    logger.Error(err)
}

// Log a message
logger.Log("This is a log message")

// Set log level
logger.SetLevel(inslogger.Info)
```

## Interface
The package provides an Interface that exposes various logging methods:
```go
type Interface interface {
    LogMultiple(errs []error)
    Log(i interface{})
    Logf(format string, args ...interface{})
    Warn(i interface{})
    Warnf(format string, args ...interface{})
    Error(err error)
    Errorf(format string, args ...interface{})
    Debug(i interface{})
    Debugf(format string, args ...interface{})
    Fatal(err error)
    Fatalf(format string, args ...interface{})
    initLogger() error
    SetLevel(level LogLevel)
}
```


# Configuration

The `inslogger` package offers configuration options to customize logging behavior.

## Log Levels

The package supports the following log levels:

- `DEBUG`
- `INFO`
- `WARN`
- `ERROR`
- `FATAL`

## Setting Log Level

The default log level is `INFO`. To set a different log level:

```go
logger := inslogger.NewLogger(inslogger.Debug)
```

## Example 
```go
// Initialize a logger with Debug level
logger := inslogger.NewLogger(inslogger.Debug)

// Log an error
err := someFunction()
if err != nil {
    logger.Error(err)
}

// Log a debug message
logger.Debugf("Debug message with arguments: %s", arg)

// Change log level to Info
logger.SetLevel(inslogger.Info)

// Log an informational message
logger.Infof("Informational message")

```
## Contribution
Feel free to contribute by forking this repository and creating pull requests. Please ensure to adhere to the existing code style and conventions.

