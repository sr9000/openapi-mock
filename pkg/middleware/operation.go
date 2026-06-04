package middleware

import (
	"context"
	"net/http"

	"openapi-mock/pkg/observability"
)

// StrictHandlerFunc matches the generated StrictHandlerFunc type from oapi-codegen v2.7+.
type StrictHandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error)

// StrictMiddlewareFunc matches the generated StrictMiddlewareFunc type from oapi-codegen v2.7+.
type StrictMiddlewareFunc func(f StrictHandlerFunc, operationID string) StrictHandlerFunc

// OperationContext annotates strict-handler context with OpenAPI operation name.
func OperationContext() StrictMiddlewareFunc {
	return func(next StrictHandlerFunc, operationID string) StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			ctx = observability.WithRequestMetadata(ctx, observability.EnsureRequestMetadata(ctx))
			ctx = observability.WithOperation(ctx, operationID)
			return next(ctx, w, r, request)
		}
	}
}
