package observability

import (
	"context"

	"github.com/rs/zerolog"
)

type requestIDKey struct{}
type traceIDKey struct{}
type operationKey struct{}
type loggerKey struct{}
type requestMetadataKey struct{}

type RequestMetadata struct {
	RequestID string
	TraceID   string
	Operation string
}

func EnsureRequestMetadata(ctx context.Context) *RequestMetadata {
	if md, ok := ctx.Value(requestMetadataKey{}).(*RequestMetadata); ok && md != nil {
		return md
	}
	return &RequestMetadata{}
}

func WithRequestMetadata(ctx context.Context, metadata *RequestMetadata) context.Context {
	if metadata == nil {
		metadata = &RequestMetadata{}
	}
	return context.WithValue(ctx, requestMetadataKey{}, metadata)
}

func RequestMetadataFromContext(ctx context.Context) *RequestMetadata {
	md, _ := ctx.Value(requestMetadataKey{}).(*RequestMetadata)
	return md
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if md := RequestMetadataFromContext(ctx); md != nil {
		md.RequestID = requestID
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	if md := RequestMetadataFromContext(ctx); md != nil && md.RequestID != "" {
		return md.RequestID
	}
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if md := RequestMetadataFromContext(ctx); md != nil {
		md.TraceID = traceID
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

func TraceID(ctx context.Context) string {
	if md := RequestMetadataFromContext(ctx); md != nil && md.TraceID != "" {
		return md.TraceID
	}
	v, _ := ctx.Value(traceIDKey{}).(string)
	return v
}

func WithOperation(ctx context.Context, operation string) context.Context {
	if md := RequestMetadataFromContext(ctx); md != nil {
		md.Operation = operation
	}
	return context.WithValue(ctx, operationKey{}, operation)
}

func Operation(ctx context.Context) string {
	if md := RequestMetadataFromContext(ctx); md != nil && md.Operation != "" {
		return md.Operation
	}
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
