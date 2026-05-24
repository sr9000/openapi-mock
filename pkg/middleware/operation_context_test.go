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
	echostub "openapi-mock/internal/stubs/echo"
	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/recorder"
)

func TestOperationContext_LabelsSuccessfulRequests(t *testing.T) {
	m := metrics.NewHTTP("0")
	rec := recorder.New()
	errHandlers := NewErrorHandlers(m, nil)

	echoHandlers := echostub.NewEchoHandlers(false)
	statusHandlers := echostub.NewStatusHandlers(false)
	strict := echostub.NewCompositeHandlers(echoHandlers, statusHandlers)
	server := echogen.NewStrictHandlerWithOptions(strict, []echogen.StrictMiddlewareFunc{OperationContext()}, echogen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandlers.RequestErrorHandler,
		ResponseErrorHandlerFunc: errHandlers.ResponseErrorHandler,
	})

	r := chi.NewRouter()
	r.Use(Recording(rec, m, RecordingOptions{BaseLogger: zerolog.Nop()}))
	echogen.HandlerFromMux(server, r)

	cases := []struct {
		name      string
		method    string
		url       string
		route     string
		body      string
		wantCode  int
		operation string
	}{
		{name: "echo", method: http.MethodPost, url: "http://example.com/echo", route: "/echo", body: `{"message":"hello"}`, wantCode: http.StatusOK, operation: "Echo"},
		{name: "echo_path", method: http.MethodGet, url: "http://example.com/echo/hello", route: "/echo/{message}", wantCode: http.StatusOK, operation: "EchoPath"},
		{name: "status", method: http.MethodGet, url: "http://example.com/status", route: "/status", wantCode: http.StatusOK, operation: "GetStatus"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpReq := httptest.NewRequest(tc.method, tc.url, http.NoBody)
			if tc.body != "" {
				httpReq = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
				httpReq.Header.Set("Content-Type", "application/json")
			}
			req := httptest.NewRecorder()
			r.ServeHTTP(req, httpReq)
			if req.Code != tc.wantCode {
				t.Fatalf("expected status %d, got %d", tc.wantCode, req.Code)
			}

			counter, err := m.HTTPRequestsTotal.GetMetricWithLabelValues(tc.method, tc.route, tc.operation, "200")
			if err != nil {
				t.Fatalf("missing metric for operation %s: %v", tc.operation, err)
			}
			if got := testutil.ToFloat64(counter); got != 1 {
				t.Fatalf("expected request counter to be 1 for operation %s, got %v", tc.operation, got)
			}
		})
	}
}
