package middleware

import (
	"net/http"

	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/observability"
)

// ContextValues injects request-id scoped values into request context.
func ContextValues(store *mm.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if store == nil {
				next.ServeHTTP(w, r)
				return
			}

			reqID := observability.RequestID(r.Context())
			ctx := mm.WithValues(r.Context(), store.Get(reqID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
