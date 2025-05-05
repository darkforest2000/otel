package handler

import (
	"fmt"
	"net/http"
	"time"
	"wapp/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func (h *Handler) wrapHandler(handler http.HandlerFunc, pattern string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		r.Response = &http.Response{}

		mainName := fmt.Sprintf("HTTP %s %s", r.Method, pattern)

		// Создаем трейс-логгер с контекстом запроса и используем более информативное имя
		tracer := otel.Tracer("http.server")
		ctx, span := tracer.Start(r.Context(), mainName)
		defer span.End()

		// Добавляем больше атрибутов для возможности фильтрации
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.target", r.URL.Path),
			attribute.String("http.remote_addr", r.RemoteAddr),
		)

		lg := logger.NewTraceLogger(ctx, "handler")

		// Используем трейс-логгер вместо обычного
		lg.Printf("Started ",
			logger.String("http.method", r.Method),
			logger.String("http.url", r.URL.String()),
			logger.String("http.remote_addr", r.RemoteAddr),
		)

		// Создаем обертку для отслеживания статус-кода
		tracker := &statusCodeTracker{ResponseWriter: w, statusCode: http.StatusOK}

		// Вызываем исходный обработчик с отслеживанием статус-кода
		handler(tracker, r.WithContext(ctx))
		// Логируем завершение запроса
		elapsed := time.Since(start)
		span.SetAttributes(
			attribute.String("http.duration_ms", fmt.Sprintf("%f", float64(elapsed.Milliseconds()))),
			attribute.Int("http.status_code", tracker.statusCode),
		)
		lg.Printf("Completed ",
			logger.String("http.url", r.URL.Path),
			logger.String("http.duration_ms", fmt.Sprintf("%f", float64(elapsed.Milliseconds()))),
			logger.Int("http.status_code", tracker.statusCode),
		)
	})
}
