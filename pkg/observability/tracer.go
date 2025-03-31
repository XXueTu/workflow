package observability

import "context"

type Span interface {
	End()
}

type Tracer interface {
	StartSpan(ctx context.Context, name string) Span
}

type defaultTracer struct{}

func NewTracer() Tracer {
	return &defaultTracer{}
}

func (t *defaultTracer) StartSpan(ctx context.Context, name string) Span {
	return &defaultSpan{}
}

type defaultSpan struct{}

func (s *defaultSpan) End() {}
