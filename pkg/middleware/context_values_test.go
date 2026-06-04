package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/recorder"
)

func TestContextValuesMiddlewareInjectsPerRequestID(t *testing.T) {
	rec := recorder.New()
	store := mm.NewStore()
	store.Replace("case-low", map[string]any{"low": 10})
	store.Replace("case-high", map[string]any{"low": 50})

	r := chi.NewRouter()
	r.Use(Recording(rec, nil, RecordingOptions{
		RequestIDHeaders:        []string{"X-Request-ID"},
		RequestIDResponseHeader: "X-Request-ID",
		BaseLogger:              zerolog.Nop(),
	}))
	r.Use(ContextValues(store))
	r.Get("/check", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%v", mm.FromCtx(r.Context(), "low"))
	})

	reqLow := httptest.NewRequest(http.MethodGet, "http://example.com/check", nil)
	reqLow.Header.Set("X-Request-ID", "case-low")
	resLow := httptest.NewRecorder()
	r.ServeHTTP(resLow, reqLow)
	if got := resLow.Body.String(); got != "10" {
		t.Fatalf("expected case-low to get 10, got %q", got)
	}

	reqHigh := httptest.NewRequest(http.MethodGet, "http://example.com/check", nil)
	reqHigh.Header.Set("X-Request-ID", "case-high")
	resHigh := httptest.NewRecorder()
	r.ServeHTTP(resHigh, reqHigh)
	if got := resHigh.Body.String(); got != "50" {
		t.Fatalf("expected case-high to get 50, got %q", got)
	}
}
