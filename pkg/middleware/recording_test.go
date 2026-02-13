package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/recorder"
)

func TestRouteTemplateFromRequest_PrefersChiRoutePattern(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/pets/{petId}", func(w http.ResponseWriter, r *http.Request) {
		got := routeTemplateFromRequest(r)
		if got != "/pets/{petId}" {
			t.Fatalf("expected route template /pets/{petId}, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/pets/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestRecording_MetricsUseRouteTemplate(t *testing.T) {
	m := metrics.NewHTTP("0")
	rec := recorder.New()

	r := chi.NewRouter()
	r.Use(Recording(rec, m, false))
	r.Get("/pets/{petId}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/pets/999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}

	ctrTpl, err := m.HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/pets/{petId}", "204")
	if err != nil {
		t.Fatalf("failed to get counter for templated labels: %v", err)
	}
	tplVal := testutil.ToFloat64(ctrTpl)

	ctrRaw, _ := m.HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/pets/999", "204")
	rawVal := 0.0
	if ctrRaw != nil {
		rawVal = testutil.ToFloat64(ctrRaw)
	}

	if tplVal != 1 {
		t.Fatalf("expected metrics recorded under templated path; got templated=%v raw=%v", tplVal, rawVal)
	}
}
