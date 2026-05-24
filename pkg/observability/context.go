package observability

import (
	"context"

	"github.com/rs/zerolog"
)

type requestIDKey struct{}
type traceIDKey struct{}
type operationKey struct{}
type loggerKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey{}).(string)
	return v
}

func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationKey{}, operation)
}

func Operation(ctx context.Context) string {
	v, _ := ctx.Value(operationKey{}).(string)
	return v
}

func WithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func Logger(ctx context.Context, fallback zerolog.Logger) zerolog.Logger {
	v := ctx.Value(loggerKey{})
	if l, ok := v.(zerolog.Logger); ok {
		return l
	}
	return fallback
}
