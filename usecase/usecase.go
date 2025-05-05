package usecase

import (
	"context"
	"fmt"
	"wapp/logger"
	"wapp/storage"
	"wapp/tractx"

	"go.opentelemetry.io/otel/attribute"
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
		lg(ctx).Error("want500", logger.Err(err))
		return "", err
	}
	return name, nil
}
