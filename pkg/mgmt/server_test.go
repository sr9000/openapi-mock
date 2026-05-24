package mgmt

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"openapi-mock/pkg/recorder"
	"strings"
	"testing"
	"time"
)

func TestHandleLogs(t *testing.T) {
	rec := recorder.New()
	rec.Record(recorder.CallRecord{
		RequestID:  "test-id",
		Method:     "/TestService/TestMethod",
		Timestamp:  time.Now(),
		Request:    map[string]string{"message": "hello"},
		Response:   map[string]string{"message": "world"},
		DurationMs: 50,
	})

	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	w := httptest.NewRecorder()

	s.handleLogs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	var records []recorder.CallRecord
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].RequestID != "test-id" {
		t.Errorf("Expected request_id 'test-id', got '%s'", records[0].RequestID)
	}
}

func TestHandleLogsEmpty(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	w := httptest.NewRecorder()

	s.handleLogs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "[]" {
		t.Errorf("Expected empty array '[]', got '%s'", string(body))
	}
}

func TestHandleLogsMethodNotAllowed(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/logs", nil)
		w := httptest.NewRecorder()

		s.handleLogs(w, req)

		resp := w.Result()
		resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for %s, got %d", method, resp.StatusCode)
		}
	}
}

func TestHandleLogsDelete(t *testing.T) {
	rec := recorder.New()
	rec.Record(recorder.CallRecord{
		RequestID: "test-id",
		Method:    "/TestService/TestMethod",
		Timestamp: time.Now(),
	})

	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodDelete, "/logs", nil)
	w := httptest.NewRecorder()

	s.handleLogs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["status"] != "cleared" {
		t.Errorf("Expected status 'cleared', got '%s'", result["status"])
	}

	if len(rec.GetRecords()) != 0 {
		t.Error("Expected records to be cleared")
	}
}

func TestHandleLogsByRequestID(t *testing.T) {
	rec := recorder.New()
	rec.Record(recorder.CallRecord{RequestID: "id-1", Method: "GET /a", Timestamp: time.Now()})
	rec.Record(recorder.CallRecord{RequestID: "id-2", Method: "GET /b", Timestamp: time.Now()})
	rec.Record(recorder.CallRecord{RequestID: "id-1", Method: "GET /c", Timestamp: time.Now()})
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/logs/id-1", nil)
	w := httptest.NewRecorder()
	s.handleLogsByRequestID(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var records []recorder.CallRecord
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}
	for _, record := range records {
		if record.RequestID != "id-1" {
			t.Fatalf("Expected only id-1 records, got %s", record.RequestID)
		}
	}
}

func TestHandleLogsByRequestIDNotFound(t *testing.T) {
	rec := recorder.New()

	s := New(rec, "9000")
	req := httptest.NewRequest(http.MethodGet, "/logs/", nil)
	w := httptest.NewRecorder()
	s.handleLogsByRequestID(w, req)

	resp := w.Result()
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestRemovedClearRouteReturnsNotFound(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "0")
	mux := http.NewServeMux()
	mux.HandleFunc("/logs", s.handleLogs)
	mux.HandleFunc("/logs/", s.handleLogsByRequestID)
	mux.HandleFunc("/doc", s.handleDoc)
	mux.HandleFunc("/openapi.json", s.handleOpenAPI)
	mux.HandleFunc("/swagger-ui-bundle.js", s.handleSwaggerUIBundle)
	mux.HandleFunc("/swagger-ui.css", s.handleSwaggerUICSS)

	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		req := httptest.NewRequest(method, "/clear", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Expected status 404 for %s /clear, got %d", method, w.Code)
		}
	}
}

func TestHandleDoc(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/doc", nil)
	w := httptest.NewRecorder()

	s.handleDoc(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got '%s'", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !contains(bodyStr, "swagger-ui") {
		t.Error("Expected response to contain 'swagger-ui'")
	}

	if !contains(bodyStr, "SwaggerUIBundle") {
		t.Error("Expected response to contain 'SwaggerUIBundle'")
	}
}

func TestHandleDocMethodNotAllowed(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodPost, "/doc", nil)
	w := httptest.NewRecorder()

	s.handleDoc(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleOpenAPI(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()

	s.handleOpenAPI(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("Failed to unmarshal OpenAPI spec: %v", err)
	}

	if spec["openapi"] != "3.0.3" {
		t.Errorf("Expected openapi version '3.0.3', got '%v'", spec["openapi"])
	}

	info, ok := spec["info"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'info' field in spec")
	}

	if info["title"] != "OpenAPI Mock Management API" {
		t.Errorf("Expected title 'OpenAPI Mock Management API', got '%v'", info["title"])
	}
}

func TestHandleSwaggerUIBundle(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/swagger-ui-bundle.js", nil)
	w := httptest.NewRecorder()

	s.handleSwaggerUIBundle(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/javascript" {
		t.Errorf("Expected Content-Type 'application/javascript', got '%s'", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) < 1000 {
		t.Error("Expected swagger-ui-bundle.js to be a large file")
	}
}

func TestHandleSwaggerUICSS(t *testing.T) {
	rec := recorder.New()
	s := New(rec, "9000")

	req := httptest.NewRequest(http.MethodGet, "/swagger-ui.css", nil)
	w := httptest.NewRecorder()

	s.handleSwaggerUICSS(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "text/css" {
		t.Errorf("Expected Content-Type 'text/css', got '%s'", resp.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) < 1000 {
		t.Error("Expected swagger-ui.css to be a large file")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
