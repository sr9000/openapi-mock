package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"

	echogen "openapi-mock/internal/generated/echo"
	petgen "openapi-mock/internal/generated/petstore"
	echostub "openapi-mock/internal/stubs/echo"
	petstub "openapi-mock/internal/stubs/petstore"
	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/recorder"
)

func TestRequestParseErrors_UseResolvedOperation(t *testing.T) {
	resolver := NewOperationResolver(map[string]string{
		"POST /echo":        "Echo",
		"GET /pets/{petId}": "GetPetById",
	})
	SetOperationResolver(resolver)
	t.Cleanup(func() { SetOperationResolver(nil) })

	m := metrics.NewHTTP("0")
	rec := recorder.New()
	errHandlers := NewErrorHandlers(m, resolver)

	r := chi.NewRouter()
	r.Use(Recording(rec, m, RecordingOptions{BaseLogger: zerolog.Nop(), OperationResolver: resolver}))

	echoStrict := echostub.NewCompositeHandlers(echostub.NewEchoHandlers(false), echostub.NewStatusHandlers(false))
	echoServer := echogen.NewStrictHandlerWithOptions(echoStrict, []echogen.StrictMiddlewareFunc{OperationContext()}, echogen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandlers.RequestErrorHandler,
		ResponseErrorHandlerFunc: errHandlers.ResponseErrorHandler,
	})
	echogen.HandlerWithOptions(echoServer, echogen.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: errHandlers.RequestErrorHandler})

	petStrict := petstub.NewCompositeHandlers(petstub.NewDefaultHandlers(false), petstub.NewPetsHandlers(false))
	petServer := petgen.NewStrictHandlerWithOptions(petStrict, []petgen.StrictMiddlewareFunc{OperationContext()}, petgen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandlers.RequestErrorHandler,
		ResponseErrorHandlerFunc: errHandlers.ResponseErrorHandler,
	})
	petgen.HandlerWithOptions(petServer, petgen.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: errHandlers.RequestErrorHandler})

	t.Run("invalid json on echo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/echo", strings.NewReader("{bad-json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
		metric, err := m.HTTPErrorsTotal.GetMetricWithLabelValues(http.MethodPost, "/echo", "Echo", "400", "request_parse")
		if err != nil {
			t.Fatalf("expected request_parse metric with Echo operation: %v", err)
		}
		if got := testutil.ToFloat64(metric); got != 1 {
			t.Fatalf("expected error counter 1, got %v", got)
		}
	})

	t.Run("invalid path param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/pets/not-int", http.NoBody)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
		metric, err := m.HTTPErrorsTotal.GetMetricWithLabelValues(http.MethodGet, "/pets/{petId}", "GetPetById", "400", "request_parse")
		if err != nil {
			t.Fatalf("expected request_parse metric with GetPetById operation: %v", err)
		}
		if got := testutil.ToFloat64(metric); got != 1 {
			t.Fatalf("expected error counter 1, got %v", got)
		}
	})

	t.Run("unmatched path stays unknown", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/does-not-exist", http.NoBody)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
		metric, err := m.HTTPRequestsTotal.GetMetricWithLabelValues(http.MethodGet, "/does-not-exist", "unknown", "404")
		if err != nil {
			t.Fatalf("expected unknown operation metric on unmatched route: %v", err)
		}
		if got := testutil.ToFloat64(metric); got != 1 {
			t.Fatalf("expected request counter 1, got %v", got)
		}
	})
}
