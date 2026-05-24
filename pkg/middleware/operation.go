package middleware

import (
	"context"
	"net/http"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"

	"openapi-mock/pkg/observability"
)

// OperationContext annotates strict-handler context with OpenAPI operation name.
func OperationContext() strictnethttp.StrictHTTPMiddlewareFunc {
	return func(next strictnethttp.StrictHTTPHandlerFunc, operationID string) strictnethttp.StrictHTTPHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			ctx = observability.WithRequestMetadata(ctx, observability.EnsureRequestMetadata(ctx))
			ctx = observability.WithOperation(ctx, operationID)
			return next(ctx, w, r, request)
		}
	}
}
