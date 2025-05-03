package logger

import (
	"context"
	"fmt"
	"io"
	"log"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracingLogger интерфейс для логирования с трейсингом
type TracingLogger interface {
	Printf(format string, v ...interface{})
}

// TraceLogger логгер, который добавляет трейс-информацию к логам
type TraceLogger struct {
	ctx context.Context
	log *log.Logger
}

// NewTraceLogger создает новый логгер с трейс-контекстом
func NewTraceLogger(ctx context.Context, w io.Writer) *TraceLogger {
	return &TraceLogger{
		ctx: ctx,
		log: log.New(w, "", log.LstdFlags),
	}
}

// Printf печатает лог с трейс-информацией
func (l *TraceLogger) Printf(format string, v ...interface{}) {
	// Получаем SpanContext из контекста
	span := trace.SpanFromContext(l.ctx)

	// Добавляем trace_id и span_id к логу
	spanCtx := span.SpanContext()
	if spanCtx.IsValid() {
		traceID := spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()
		formattedMsg := fmt.Sprintf(format, v...)
		l.log.Printf("[trace_id=%s span_id=%s] %s", traceID, spanID, formattedMsg)

		// Также добавляем событие в спан
		span.AddEvent("log", trace.WithAttributes(
			attribute.String("message", formattedMsg),
		))
	} else {
		// Если контекст не содержит валидный спан, печатаем обычный лог
		l.log.Printf(format, v...)
	}
}

// DefaultLogger стандартная реализация Logger
type DefaultLogger struct{}

// NewDefaultLogger создает стандартный логгер
func NewDefaultLogger() TracingLogger {
	return &DefaultLogger{}
}

// Printf печатает лог
func (l *DefaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}
