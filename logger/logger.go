package logger

import (
	"context"
	"fmt"
	logstd "log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

type KeyValue struct {
	log.KeyValue
}

func Err(err error) KeyValue {
	val := ""
	if err != nil {
		val = err.Error()
	}
	return KeyValue{log.String("error", val)}
}

func String(key, value string) KeyValue {
	return KeyValue{log.String(key, value)}
}

func Int[v int | int8 | int16 | int32 | int64](key string, value v) KeyValue {
	return KeyValue{log.Int(key, int(value))}
}

func Float[v float32 | float64](key string, value v) KeyValue {
	return KeyValue{log.Float64(key, float64(value))}
}

func Bool(key string, value bool) KeyValue {
	return KeyValue{log.Bool(key, value)}
}

func Bytes(key string, value []byte) KeyValue {
	return KeyValue{log.Bytes(key, value)}
}

// TracingLogger интерфейс для логирования с трейсингом
type TracingLogger interface {
	Printf(format string, v ...interface{})
}

// TraceLogger логгер, который добавляет трейс-информацию к логам
type TraceLogger struct {
	ctx  context.Context
	log  *logstd.Logger
	name string
}

// NewTraceLogger создает новый логгер с трейс-контекстом
func NewTraceLogger(ctx context.Context, name string) *TraceLogger {
	return &TraceLogger{
		ctx:  ctx,
		log:  logstd.New(os.Stdout, "", logstd.LstdFlags),
		name: name,
	}
}

// Printf печатает лог с трейс-информацией
func (l *TraceLogger) Printf(format string, attrs ...KeyValue) {
	l.logHelper(l.ctx, log.SeverityInfo, "INFO", format, attrs...)
}

func (l *TraceLogger) Info(msg string, attrs ...KeyValue) {
	l.logHelper(l.ctx, log.SeverityInfo, "INFO", msg, attrs...)
}

func (l *TraceLogger) Error(msg string, attrs ...KeyValue) {
	l.logHelper(l.ctx, log.SeverityError, "ERROR", msg, attrs...)
}

func (l *TraceLogger) Debug(msg string, attrs ...KeyValue) {
	l.logHelper(l.ctx, log.SeverityDebug, "DEBUG", msg, attrs...)
}

func (l *TraceLogger) Warn(msg string, attrs ...KeyValue) {
	l.logHelper(l.ctx, log.SeverityWarn, "WARN", msg, attrs...)
}

func (l *TraceLogger) Fatal(msg string, attrs ...KeyValue) {
	l.logHelper(l.ctx, log.SeverityFatal, "FATAL", msg, attrs...)
}

func buildMsg(msg string, attrs ...KeyValue) string {
	var buf strings.Builder
	buf.WriteString(msg)
	for _, attr := range attrs {
		buf.WriteString(fmt.Sprintf(": %s=%s", attr.Key, attr.Value))
	}
	return buf.String()
}

func (l *TraceLogger) logHelper(ctx context.Context, severity log.Severity, severityText string, msg string, attrs ...KeyValue) {
	logger := global.GetLoggerProvider().Logger("main") // Or derive logger name from call stack if needed
	record := log.Record{}
	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetSeverityText(severityText)
	record.SetBody(log.StringValue(msg))

	// Add trace context if available
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()
	if spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()
		formattedMsg := buildMsg(msg, attrs...)
		l.log.Printf("[trace_id=%s span_id=%s] %s", traceID, spanID, formattedMsg)

		// Также добавляем событие в спан
		span.AddEvent("log", trace.WithAttributes(
			attribute.String("message", formattedMsg),
		))
	} else {
		// Если контекст не содержит валидный спан, печатаем обычный лог
		l.log.Printf(msg)
	}

	// Add extra attributes
	if len(attrs) > 0 {
		attrs := make([]log.KeyValue, len(attrs))
		for i, attr := range attrs {
			attrs[i] = log.KeyValue(attr)
		}
		record.AddAttributes(attrs...)
	}

	logger.Emit(ctx, record)
}

// DefaultLogger стандартная реализация Logger
type DefaultLogger struct{}

// NewDefaultLogger создает стандартный логгер
func NewDefaultLogger() TracingLogger {
	return &DefaultLogger{}
}

// Printf печатает лог
func (l *DefaultLogger) Printf(format string, v ...interface{}) {
	logstd.Printf(format, v...)
}
