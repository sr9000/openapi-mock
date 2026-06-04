package mgmt

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"openapi-mock/pkg/middleware"
	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/recorder"
)

func TestContextValuesByRequestID_EndToEnd(t *testing.T) {
	rec := recorder.New()
	store := mm.NewStore()

	mgmtSrv := New(Options{Recorder: rec, ContextValues: store, Port: "9000"})
	mgmtTS := httptest.NewServer(mgmtSrv.router())
	defer mgmtTS.Close()

	r := chi.NewRouter()
	r.Use(middleware.Recording(rec, nil, middleware.RecordingOptions{BaseLogger: zerolog.Nop()}))
	r.Use(middleware.ContextValues(store))
	r.Get("/check", func(w http.ResponseWriter, r *http.Request) {
		v := 0
		_, _ = fmt.Sscanf(r.URL.Query().Get("val"), "%d", &v)

		low, _ := mm.Lookup(r.Context(), "low")
		high, _ := mm.Lookup(r.Context(), "high")
		lowInt, _ := low.(int)
		highInt, _ := high.(int)

		if v < lowInt {
			http.Error(w, "not enough", http.StatusBadRequest)
			return
		}
		if v > highInt {
			http.Error(w, "too much", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	mockTS := httptest.NewServer(r)
	defer mockTS.Close()

	mustRequest(t, http.MethodPut, mgmtTS.URL+"/context-values/case-low", `{"low":10,"high":20}`, "")
	mustRequest(t, http.MethodPut, mgmtTS.URL+"/context-values/case-high", `{"low":50,"high":100}`, "")

	statusLow, bodyLow := mustRequest(t, http.MethodGet, mockTS.URL+"/check?val=15", "", "case-low")
	if statusLow != http.StatusOK || bodyLow != "ok" {
		t.Fatalf("case-low expected 200/ok, got %d/%q", statusLow, bodyLow)
	}

	statusHigh, bodyHigh := mustRequest(t, http.MethodGet, mockTS.URL+"/check?val=15", "", "case-high")
	if statusHigh != http.StatusBadRequest || !strings.Contains(bodyHigh, "not enough") {
		t.Fatalf("case-high expected 400/not enough, got %d/%q", statusHigh, bodyHigh)
	}
}

func mustRequest(t *testing.T, method, url, body, requestID string) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(respBody))
}
