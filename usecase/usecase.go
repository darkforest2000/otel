package usecase

import (
	"context"
	"fmt"
	"wapp/logger"
	"wapp/storage"
	"wapp/tractx"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Usecase struct {
	storage *storage.Storage
}

func New(storage *storage.Storage) *Usecase {
	return &Usecase{storage: storage}
}

func lg(ctx context.Context) *logger.TraceLogger {
	return logger.NewTraceLogger(ctx, "usecase")
}

func (u *Usecase) Hello(ctx tractx.Context, name string) (string, error) {
	ctx, span, stop := ctx.TracerStart("helloUsecase")
	defer stop()

	span.SetAttributes(attribute.String("nameInUsecase", name))

	if name == "want500" {
		err := fmt.Errorf("want500")
		lg(ctx).Error("want500", logger.Err(err))
		return "", err
	}

	name, err := u.storage.GetSurname(ctx, name)
	if err != nil {
		lg(ctx).Error("storage error", logger.Err(err))
		return "", err
	}

	// Получаем Meter
	meter := otel.GetMeterProvider().Meter("usecase-hello")

	// Создаем Counter
	counter, err := meter.Int64Counter(
		"processed_names_total", // Новое имя метрики
		metric.WithDescription("Total number of names processed by Hello usecase"),
		metric.WithUnit("{names}"), // Указываем единицу измерения (опционально)
	)
	if err != nil {
		lg(ctx).Error("error creating counter", logger.Err(err)) // Логируем ошибку
		return "", err 
	}

	// Увеличиваем счетчик на 1 с атрибутом
	counter.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(attribute.String("name_processed", name))))
	return name, nil
}
