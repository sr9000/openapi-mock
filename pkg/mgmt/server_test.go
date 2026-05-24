package mgmt

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/recorder"
)

func TestLogsRoutes(t *testing.T) {
	rec := recorder.New()
	rec.Record(recorder.CallRecord{RequestID: "id-1", Method: "GET /a", Timestamp: time.Now()})
	rec.Record(recorder.CallRecord{RequestID: "id-2", Method: "GET /b", Timestamp: time.Now()})
	s := New(rec, mm.NewStore(), "9000")
	h := s.router()

	getReq := httptest.NewRequest(http.MethodGet, "/logs", nil)
	getRes := httptest.NewRecorder()
	h.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRes.Code)
	}

	filterReq := httptest.NewRequest(http.MethodGet, "/logs/id-1", nil)
	filterRes := httptest.NewRecorder()
	h.ServeHTTP(filterRes, filterReq)
	if filterRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", filterRes.Code)
	}
	var filtered []recorder.CallRecord
	if err := json.Unmarshal(filterRes.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(filtered) != 1 || filtered[0].RequestID != "id-1" {
		t.Fatalf("unexpected filtered records: %+v", filtered)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/logs", nil)
	deleteRes := httptest.NewRecorder()
	h.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteRes.Code)
	}
	if len(rec.GetRecords()) != 0 {
		t.Fatalf("expected logs to be cleared")
	}
}

func TestContextValuesCollectionEndpoints(t *testing.T) {
	store := mm.NewStore()
	s := New(recorder.New(), store, "9000")
	h := s.router()

	put := httptest.NewRequest(http.MethodPut, "/context-values", strings.NewReader(`{"case-a":{"low":10},"case-b":{"high":20.5}}`))
	put.Header.Set("Content-Type", "application/json")
	putRes := httptest.NewRecorder()
	h.ServeHTTP(putRes, put)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", putRes.Code)
	}

	patch := httptest.NewRequest(http.MethodPatch, "/context-values", strings.NewReader(`{"case-a":{"high":30}}`))
	patch.Header.Set("Content-Type", "application/json")
	patchRes := httptest.NewRecorder()
	h.ServeHTTP(patchRes, patch)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", patchRes.Code)
	}
	if store.Get("case-a")["high"] != int(30) {
		t.Fatalf("expected merged high=30, got %#v", store.Get("case-a")["high"])
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/context-values", nil)
	deleteRes := httptest.NewRecorder()
	h.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteRes.Code)
	}
	if len(store.GetAll()) != 0 {
		t.Fatalf("expected store to be empty")
	}
}

func TestContextValuesRequestIDEndpoints(t *testing.T) {
	store := mm.NewStore()
	s := New(recorder.New(), store, "9000")
	h := s.router()

	put := httptest.NewRequest(http.MethodPut, "/context-values/case-a", strings.NewReader(`{"low":10,"high":20}`))
	putRes := httptest.NewRecorder()
	h.ServeHTTP(putRes, put)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", putRes.Code)
	}

	patch := httptest.NewRequest(http.MethodPatch, "/context-values/case-a", strings.NewReader(`{"high":30}`))
	patchRes := httptest.NewRecorder()
	h.ServeHTTP(patchRes, patch)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", patchRes.Code)
	}

	deleteKeyReq := httptest.NewRequest(http.MethodDelete, "/context-values/case-a", strings.NewReader(`{"keys":["low"]}`))
	deleteKeyRes := httptest.NewRecorder()
	h.ServeHTTP(deleteKeyRes, deleteKeyReq)
	if deleteKeyRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteKeyRes.Code)
	}
	if _, ok := store.Get("case-a")["low"]; ok {
		t.Fatalf("expected low to be deleted")
	}

	deleteAllReq := httptest.NewRequest(http.MethodDelete, "/context-values/case-a", nil)
	deleteAllRes := httptest.NewRecorder()
	h.ServeHTTP(deleteAllRes, deleteAllReq)
	if deleteAllRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteAllRes.Code)
	}
	if len(store.Get("case-a")) != 0 {
		t.Fatalf("expected case-a to be removed")
	}
}

func TestContextValuesInvalidJSONAndUnknownRequestID(t *testing.T) {
	s := New(recorder.New(), mm.NewStore(), "9000")
	h := s.router()

	bad := httptest.NewRequest(http.MethodPut, "/context-values/case-a", strings.NewReader("{"))
	badRes := httptest.NewRecorder()
	h.ServeHTTP(badRes, bad)
	if badRes.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", badRes.Code)
	}

	unknown := httptest.NewRequest(http.MethodGet, "/context-values/missing", nil)
	unknownRes := httptest.NewRecorder()
	h.ServeHTTP(unknownRes, unknown)
	if unknownRes.Code != http.StatusOK || strings.TrimSpace(unknownRes.Body.String()) != "{}" {
		t.Fatalf("expected unknown request id response '{}', got status=%d body=%q", unknownRes.Code, unknownRes.Body.String())
	}
}

func TestManagementRouteMethodsAndDocs(t *testing.T) {
	s := New(recorder.New(), mm.NewStore(), "9000")
	h := s.router()

	req405 := httptest.NewRequest(http.MethodPost, "/context-values", nil)
	res405 := httptest.NewRecorder()
	h.ServeHTTP(res405, req405)
	if res405.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res405.Code)
	}

	clearReq := httptest.NewRequest(http.MethodPost, "/clear", nil)
	clearRes := httptest.NewRecorder()
	h.ServeHTTP(clearRes, clearReq)
	if clearRes.Code != http.StatusNotFound {
		t.Fatalf("expected /clear to be 404, got %d", clearRes.Code)
	}

	docReq := httptest.NewRequest(http.MethodGet, "/doc", nil)
	docRes := httptest.NewRecorder()
	h.ServeHTTP(docRes, docReq)
	if docRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", docRes.Code)
	}
	if !strings.Contains(docRes.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("expected swagger ui html")
	}

	openapiReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openapiRes := httptest.NewRecorder()
	h.ServeHTTP(openapiRes, openapiReq)
	if openapiRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", openapiRes.Code)
	}
	body, _ := io.ReadAll(openapiRes.Body)
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("invalid openapi json: %v", err)
	}
}
