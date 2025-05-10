package usecase

import (
	"context"
	"fmt"
	"wapp/logger"
	"wapp/storage"
	"wapp/tractx"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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
		return "", handleError(
			ctx,
			span,
			"simulated usecase error for want500",
			fmt.Errorf("simulated usecase error for want500"),
		)
	}

	resultName, err := u.storage.GetSurname(ctx, name)
	if err != nil {
		return "", handleError(
			ctx,
			span,
			"storage error in usecase",
			err,
		)
	}

	meter := otel.GetMeterProvider().Meter("usecase-hello")
	counter, mErr := meter.Int64Counter(
		"processed_names_total",
		metric.WithDescription("Total number of names processed by Hello usecase"),
		metric.WithUnit("{names}"),
	)
	if mErr != nil {
		lg(ctx).Error("error creating counter in usecase", logger.Err(mErr))
		span.RecordError(fmt.Errorf("metric counter creation failed: %w", mErr))
	} else {
		counter.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(attribute.String("name_processed", resultName))))
	}

	span.SetStatus(codes.Ok, "")
	return resultName, nil
}

func handleError(ctx context.Context, span trace.Span, msg string, err error) error {
	lg(ctx).Error(msg, logger.Err(err))
	span.RecordError(err, trace.WithStackTrace(true))
	span.SetStatus(codes.Error, msg)
	return err
}
