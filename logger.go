package main

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"sync"
// 	"time"

// 	"github.com/sirupsen/logrus"
// 	"gitlab.corp.mail.ru/calendar/hardgo/v2/errors"
// )

// type Level string

// const (
// 	LevelError Level = "error"
// 	LevelWarn  Level = "warn"
// 	LevelInfo  Level = "info"
// 	LevelDebug Level = "debug"
// )

// var configLevels = map[Level]logrus.Level{
// 	LevelError: logrus.ErrorLevel,
// 	LevelWarn:  logrus.WarnLevel,
// 	LevelInfo:  logrus.InfoLevel,
// 	LevelDebug: logrus.DebugLevel,
// }

// var logrusLevels = map[logrus.Level]Level{
// 	logrus.ErrorLevel: LevelError,
// 	logrus.WarnLevel:  LevelWarn,
// 	logrus.InfoLevel:  LevelInfo,
// 	logrus.DebugLevel: LevelDebug,
// }

// type Fields map[string]interface{}

// func (f Fields) Extend(f2 Fields) {
// 	for k, v := range f2 {
// 		f[k] = v
// 	}
// }

// func (f Fields) Copy() Fields {
// 	ret := make(Fields, len(f))
// 	for k, v := range f {
// 		ret[k] = v
// 	}
// 	return ret
// }

// type Logger interface {
// 	ForCtx(ctx context.Context) Logger
// 	WithField(key string, val interface{}) Logger
// 	WithFields(fields Fields) Logger
// 	WithError(err error) Logger
// 	Debug(args ...interface{})
// 	Debugf(format string, args ...interface{})
// 	Info(args ...interface{})
// 	Infof(format string, args ...interface{})
// 	Warn(args ...interface{})
// 	Warnf(format string, args ...interface{})
// 	Error(args ...interface{})
// 	Errorf(format string, args ...interface{})
// 	Fatal(args ...interface{})
// 	Fatalf(format string, args ...interface{})
// 	Print(args ...interface{})
// 	Printf(format string, args ...interface{})
// 	Log(level Level, args ...interface{})
// 	Logf(level Level, format string, args ...interface{})
// 	Level() Level
// }

// type Config struct {
// 	App   string `mapstructure:"app"`
// 	Level Level  `mapstructure:"level"`
// }

// func (c Config) Validate() error {
// 	if c.App == "" {
// 		return errors.New("empty app")
// 	}

// 	if _, ok := configLevels[c.Level]; !ok {
// 		return errors.Errorf("unknown log level: %q", c.Level)
// 	}

// 	return nil
// }

// func NewLogrusLogger(cfg Config) (Logger, error) {
// 	if err := cfg.Validate(); err != nil {
// 		return nil, err
// 	}

// 	logrusLogger := logrus.New()
// 	logrusLogger.SetLevel(configLevels[cfg.Level])
// 	logrusLogger.SetFormatter(&logrus.JSONFormatter{
// 		TimestampFormat: time.RFC3339Nano,
// 	})

// 	l := logrusLogger.WithField("app", cfg.App)

// 	hostname, err := os.Hostname()
// 	if err == nil {
// 		l = l.WithField("host", hostname)
// 	}

// 	return logger{Entry: l}, nil
// }

// var def Logger

// func init() {
// 	var err error
// 	def, err = NewLogrusLogger(Config{
// 		App:   "initializing",
// 		Level: "debug",
// 	})
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to init default logger: %s", err))
// 	}
// }

// func Default() Logger {
// 	return def
// }

// type logger struct {
// 	*logrus.Entry
// }

// func (l logger) ForCtx(ctx context.Context) Logger {
// 	if isDebugLoggingEnabled(ctx) {
// 		l.Entry.Level = logrus.DebugLevel
// 	}

// 	return l.WithFields(GetCtxFields(ctx))
// }

// func isDebugLoggingEnabled(ctx context.Context) bool {
// 	return true
// }

// func (l logger) WithFields(fields Fields) Logger {
// 	if fields == nil {
// 		return l
// 	}

// 	return logger{Entry: l.Entry.WithFields(logrus.Fields(fields))}
// }

// func (l logger) WithField(key string, val interface{}) Logger {
// 	return logger{Entry: l.Entry.WithField(key, val)}
// }

// func (l logger) WithError(err error) Logger {
// 	return logger{Entry: l.Entry.WithError(err)}
// }

// func (l logger) Debug(args ...interface{}) { l.Debugln(args...) }
// func (l logger) Info(args ...interface{})  { l.Infoln(args...) }
// func (l logger) Warn(args ...interface{})  { l.Warnln(args...) }
// func (l logger) Error(args ...interface{}) { l.Errorln(args...) }
// func (l logger) Fatal(args ...interface{}) { l.Fatalln(args...) }
// func (l logger) Print(args ...interface{}) { l.Println(args...) }

// func (l logger) Log(level Level, args ...interface{}) { l.Logln(configLevels[level], args...) }

// func (l logger) Logf(level Level, format string, args ...interface{}) {
// 	l.Entry.Logf(configLevels[level], format, args...)
// }

// func (l logger) Level() Level { return logrusLevels[l.Entry.Logger.Level] }

// func GetCtxFields(ctx context.Context) Fields {
// 	f, ok := getCtxFields(ctx)
// 	if !ok {
// 		return Fields{}
// 	}

// 	f.mx.RLock()
// 	defer f.mx.RUnlock()

// 	return f.mp.Copy()
// }

// func getCtxFields(ctx context.Context) (ctxFields, bool) {
// 	value := ctx.Value(fieldsCtxKey{})
// 	if value == nil {
// 		return ctxFields{}, false
// 	}

// 	return value.(ctxFields), true
// }

// type ctxFields struct {
// 	mp Fields
// 	mx *sync.RWMutex
// }

// type fieldsCtxKey struct{}
