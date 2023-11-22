package inslogger

import (
	"go.uber.org/zap"
	"log"
)

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
type LogLevel string

const (
	Debug LogLevel = "DEBUG"
	Info  LogLevel = "INFO"
	Warn  LogLevel = "WARN"
	Error LogLevel = "ERROR"
	Fatal LogLevel = "FATAL"
)

func (ll LogLevel) toZapLevel() zap.AtomicLevel {
	switch ll {
	case Debug:
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case Info:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case Warn:
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case Error:
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	case Fatal:
		return zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}

type AppLogger struct {
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger

	Level LogLevel
}

func NewLogger(level LogLevel) Interface {
	al := &AppLogger{
		Logger: nil,
		Sugar:  nil,
		Level:  level,
	}
	if err := al.initLogger(); err != nil {
		log.Fatalf("zap.Init: %+v", err)
	}

	return al
}

func NewNopLogger() Interface {
	return &AppLogger{
		Logger: zap.NewNop(),
		Sugar:  zap.NewNop().Sugar(),
	}
}

func (al *AppLogger) LogMultiple(errs []error) {
	for _, err := range errs {
		al.Sugar.Infof("%v+", err)
	}
}

func (al *AppLogger) Log(i interface{}) {
	al.Sugar.Infof("%s", i)
}

func (al *AppLogger) Logf(format string, args ...interface{}) {
	al.Sugar.Infof(format, args...)
}

func (al *AppLogger) Warn(i interface{}) {
	al.Sugar.Warnf("%s", i)
}

func (al *AppLogger) Warnf(format string, args ...interface{}) {
	al.Sugar.Warnf(format, args...)
}

func (al *AppLogger) Error(err error) {
	al.Sugar.Errorf("%v+", err)
}

func (al *AppLogger) Errorf(format string, args ...interface{}) {
	al.Sugar.Errorf(format, args...)
}

func (al *AppLogger) Debug(i interface{}) {
	al.Sugar.Debugf("%s", i)
}

func (al *AppLogger) Debugf(format string, args ...interface{}) {
	al.Sugar.Debugf(format, args...)
}

func (al *AppLogger) Fatal(err error) {
	al.Sugar.Fatalf("log.Fatal: %+v\n", err)
}

func (al *AppLogger) Fatalf(format string, args ...interface{}) {
	al.Sugar.Fatalf(format, args...)
}

func (al *AppLogger) initLogger() error {
	var (
		newLogger *zap.Logger
		err       error
	)

	newLogger, err = zap.NewProduction()
	if err != nil {
		return err
	}

	switch al.Level {
	case Debug:
		newLogger, err = zap.NewDevelopment()
		if err != nil {
			return err
		}
		newLogger = newLogger.WithOptions(zap.IncreaseLevel(zap.DebugLevel))
	case Info:
		newLogger = newLogger.WithOptions(zap.IncreaseLevel(zap.InfoLevel))
	case Warn:
		newLogger = newLogger.WithOptions(zap.IncreaseLevel(zap.WarnLevel))
	case Error:
		newLogger = newLogger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel))
	case Fatal:
		newLogger = newLogger.WithOptions(zap.IncreaseLevel(zap.FatalLevel))
	default:
		newLogger = newLogger.WithOptions(zap.IncreaseLevel(zap.InfoLevel))
	}

	al.Logger = newLogger
	al.Sugar = newLogger.Sugar()

	return nil
}

func (al *AppLogger) SetLevel(level LogLevel) {
	al.Logger.WithOptions(zap.IncreaseLevel(level.toZapLevel()))
}
