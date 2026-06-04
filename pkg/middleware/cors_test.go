package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSWildcardAllowsAnyOrigin(t *testing.T) {
	h := CORS(CORSOptions{AllowOrigins: "*"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	req.Header.Set("Origin", "http://localhost:9000")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard allow origin, got %q", got)
	}
}

func TestCORSAllowList(t *testing.T) {
	h := CORS(CORSOptions{AllowOrigins: "http://localhost:9000,http://example.test"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	allowedReq := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	allowedReq.Header.Set("Origin", "http://localhost:9000")
	allowedRes := httptest.NewRecorder()
	h.ServeHTTP(allowedRes, allowedReq)
	if got := allowedRes.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:9000" {
		t.Fatalf("expected allowed origin to be echoed, got %q", got)
	}

	disallowedReq := httptest.NewRequest(http.MethodGet, "http://example.com/x", nil)
	disallowedReq.Header.Set("Origin", "http://not-allowed")
	disallowedRes := httptest.NewRecorder()
	h.ServeHTTP(disallowedRes, disallowedReq)
	if got := disallowedRes.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected disallowed origin to be omitted, got %q", got)
	}
}

func TestCORSPreflightReturnsNoContent(t *testing.T) {
	nextCalled := false
	h := CORS(CORSOptions{AllowOrigins: "*"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://example.com/x", nil)
	req.Header.Set("Origin", "http://localhost:9000")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
	if nextCalled {
		t.Fatalf("expected preflight not to call next handler")
	}
}
