package middleware

import (
	"net/http"
	"strings"
)

type CORSOptions struct {
	AllowOrigins string
}

// CORS adds minimal CORS headers for browser-based calls from management Swagger UI.
func CORS(opts CORSOptions) func(http.Handler) http.Handler {
	origins, wildcard := parseAllowOrigins(opts.AllowOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				if wildcard {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if origins[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID,X-Request-Id,X-Correlation-ID,traceparent")
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseAllowOrigins(raw string) (map[string]bool, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "*" {
		return nil, true
	}
	parts := strings.Split(trimmed, ",")
	out := make(map[string]bool, len(parts))
	for _, p := range parts {
		origin := strings.TrimSpace(p)
		if origin == "" {
			continue
		}
		if origin == "*" {
			return nil, true
		}
		out[origin] = true
	}
	if len(out) == 0 {
		return nil, true
	}
	return out, false
}
