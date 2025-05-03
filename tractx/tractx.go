package tractx

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type Context interface {
	context.Context
	Span() trace.Span
	TracerStart(name string) (ctx Context, span trace.Span, stop func(options ...trace.SpanEndOption))
}

type icontext struct {
	context.Context
	span trace.Span
}

func New(ctx context.Context) Context {
	span := trace.SpanFromContext(ctx)
	return &icontext{Context: ctx, span: span}
}

func (c *icontext) Span() trace.Span {
	return c.span
}

func (c *icontext) TracerStart(name string) (Context, trace.Span, func(options ...trace.SpanEndOption)) {
	ctx, span := c.span.TracerProvider().Tracer(name).Start(c.Context, name)
	return &icontext{Context: ctx, span: span}, span, span.End
}
