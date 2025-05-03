package usecase

import (
	"fmt"
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

func (u *Usecase) Hello(ctx tractx.Context, name string) (string, error) {
	// ...
	ctx, span, stop := ctx.TracerStart("helloUsecase")
	defer stop()

	span.SetAttributes(attribute.String("nameInUsecase", name))

	if name == "want500" {
		err := fmt.Errorf("want500")
		span.RecordError(err)
		return "", err
	}

	name, err := u.storage.GetSurname(ctx, name)
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	return name, nil
}
