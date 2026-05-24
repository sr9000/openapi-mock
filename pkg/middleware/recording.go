package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"openapi-mock/pkg/ctxkeys"
	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/observability"
	"openapi-mock/pkg/recorder"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

type RecordingOptions struct {
	EnableLogging          bool
	RequestIDHeaders       []string
	RequestIDResponseHeader string
	BaseLogger             zerolog.Logger
	Tracer                 trace.Tracer
}

func Recording(rec *recorder.Recorder, m *metrics.Metrics, opts RecordingOptions) func(http.Handler) http.Handler {
	if len(opts.RequestIDHeaders) == 0 {
		opts.RequestIDHeaders = []string{"X-Request-ID", "X-Request-Id", "X-Correlation-ID"}
	}
	if strings.TrimSpace(opts.RequestIDResponseHeader) == "" {
		opts.RequestIDResponseHeader = observability.DefaultRequestIDResponseHeader
	}
	if opts.Tracer == nil {
		opts.Tracer = otel.Tracer("openapi-mock/http")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := observability.ResolveRequestID(r.Header.Get, opts.RequestIDHeaders)
			start := time.Now()
			traceID := observability.TraceIDFromTraceparent(r.Header.Get("traceparent"))

			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			rw := &responseWriter{ResponseWriter: w, statusCode: 200}
			rw.Header().Set(opts.RequestIDResponseHeader, reqID)

			metadata := observability.EnsureRequestMetadata(r.Context())
			ctx := observability.WithRequestMetadata(r.Context(), metadata)
			ctx = observability.WithRequestID(ctx, reqID)
			ctx = context.WithValue(ctx, ctxkeys.RequestID{}, reqID)
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
			ctx, span := opts.Tracer.Start(ctx, r.Method+" "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()
			if sc := span.SpanContext(); sc.IsValid() {
				traceID = sc.TraceID().String()
			}

			requestLogger := opts.BaseLogger.With().
				Str("request_id", reqID).
				Str("trace_id", traceID).
				Str("method", r.Method).
				Logger()

			ctx = observability.WithTraceID(ctx, traceID)
			ctx = observability.WithLogger(ctx, requestLogger)
			r = r.WithContext(ctx)

			// pathLabel is computed after the handler runs, because the router populates
			// the matched route pattern during request dispatch.
			pathLabel := ""

			defer func() {
				if pathLabel == "" {
					pathLabel = routeTemplateFromRequest(r)
				}
				operation := observability.Operation(r.Context())
				if operation == "" {
					operation = "unknown"
				}
				if err := recover(); err != nil {
					duration := time.Since(start)
					panicMsg := fmt.Sprintf("%v", err)
					span.RecordError(fmt.Errorf("panic: %s", panicMsg))
					span.SetAttributes(
						attribute.String("http.route", pathLabel),
						attribute.String("openapi.operation", operation),
						attribute.Int("http.status_code", 500),
					)
					rec.Record(recorder.CallRecord{
						RequestID:  reqID,
						Method:     r.Method + " " + pathLabel,
						Timestamp:  start,
						Request:    string(bodyBytes),
						Panic:      panicMsg,
						DurationMs: duration.Milliseconds(),
					})
					if m != nil {
						m.RecordHTTPRequest(r.Method, pathLabel, operation, duration.Milliseconds(), 500)
						m.RecordHTTPPanic(r.Method, pathLabel, operation, 500, "panic")
					}
					if opts.EnableLogging {
						requestLogger.Error().
							Str("route", pathLabel).
							Str("operation", operation).
							Int("status", 500).
							Int64("duration_ms", duration.Milliseconds()).
							Str("panic", panicMsg).
							Msg("http request panic")
					}
					http.Error(w, "Internal Server Error", 500)
				}
			}()

			if opts.EnableLogging {
				requestLogger.Info().Str("path", r.URL.Path).Msg("http request started")
			}

			next.ServeHTTP(rw, r)

			if pathLabel == "" {
				pathLabel = routeTemplateFromRequest(r)
			}

			operation := observability.Operation(r.Context())
			if operation == "" {
				operation = "unknown"
			}

			duration := time.Since(start)
			span.SetAttributes(
				attribute.String("http.route", pathLabel),
				attribute.String("openapi.operation", operation),
				attribute.Int("http.status_code", rw.statusCode),
			)

			rec.Record(recorder.CallRecord{
				RequestID:  reqID,
				Method:     r.Method + " " + pathLabel,
				Timestamp:  start,
				Request:    string(bodyBytes),
				Response:   rw.body.String(),
				DurationMs: duration.Milliseconds(),
			})

			if m != nil {
				m.RecordHTTPRequest(r.Method, pathLabel, operation, duration.Milliseconds(), rw.statusCode)
			}

			if opts.EnableLogging {
				requestLogger.Info().
					Str("route", pathLabel).
					Str("operation", operation).
					Int("status", rw.statusCode).
					Int64("duration_ms", duration.Milliseconds()).
					Msg("http request completed")
			}
		})
	}
}

func routeTemplateFromRequest(r *http.Request) string {
	// Prefer Chi's matched route pattern. oapi-codegen uses chi patterns that match OpenAPI templates.
	if r == nil {
		return ""
	}
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
		// Fallback: join all patterns (older chi versions / edge cases)
		if len(rctx.RoutePatterns) > 0 {
			return rctx.RoutePatterns[len(rctx.RoutePatterns)-1]
		}
	}
	if r.URL != nil {
		return r.URL.Path
	}
	return ""
}
