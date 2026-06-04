package mgmt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/recorder"
)

// TestAllDocumentedRoutesAreServed reads the embedded openapi.json, enumerates
// every path+method, issues a representative request for each, and asserts the
// response is NOT 404 (route missing) or 405 (method not allowed).
func TestAllDocumentedRoutesAreServed(t *testing.T) {
	// Parse the embedded OpenAPI spec.
	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(openapiSpec, &spec); err != nil {
		t.Fatalf("failed to parse embedded openapi.json: %v", err)
	}

	// Build a real management server with dummy dependencies.
	s := New(Options{
		Recorder:      recorder.New(),
		ContextValues: mm.NewStore(),
		Port:          "9000",
		MockDocs: []MockDoc{
			{APIName: "petstore", Title: "Petstore", SpecJSON: func() ([]byte, error) {
				return []byte(`{"openapi":"3.0.3","info":{"title":"Petstore"}}`), nil
			}},
			{APIName: "echo", APIVersion: "v1", Title: "Echo v1", SpecJSON: func() ([]byte, error) {
				return []byte(`{"openapi":"3.0.3","info":{"title":"Echo v1"}}`), nil
			}},
		},
		Reset: func(_ context.Context) error { return nil },
	})
	h := s.router()

	// Placeholder values for path parameters.
	pathParams := map[string]string{
		"request_id": "smoke-id",
		"api_name":   "echo",
		"api_ver":    "v1",
	}

	for path, methods := range spec.Paths {
		for method := range methods {
			resolved := resolvePathParams(path, pathParams)

			var req *http.Request
			switch method {
			case "put", "patch":
				body := `{}`
				if strings.Contains(resolved, "context-values") && !strings.Contains(resolved, "request_id") {
					body = `{"case-a":{}}`
				}
				req = httptest.NewRequest(toHTTPMethod(method), resolved, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			case "post":
				body := `{}`
				req = httptest.NewRequest(http.MethodPost, resolved, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			case "delete":
				req = httptest.NewRequest(http.MethodDelete, resolved, nil)
				// DELETE /context-values/{request_id} may have a JSON body with keys
				if strings.Contains(resolved, "context-values/") {
					req = httptest.NewRequest(http.MethodDelete, resolved, strings.NewReader(`{"keys":["k"]}`))
					req.Header.Set("Content-Type", "application/json")
				}
			default:
				req = httptest.NewRequest(toHTTPMethod(method), resolved, nil)
			}

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code == http.StatusNotFound {
				t.Errorf("%s %s → 404 (route not registered)", method, resolved)
			} else if rr.Code == http.StatusMethodNotAllowed {
				t.Errorf("%s %s → 405 (method not allowed)", method, resolved)
			}
		}
	}
}

// resolvePathParams replaces {param} placeholders with sample values.
func resolvePathParams(path string, params map[string]string) string {
	result := path
	for k, v := range params {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}

func toHTTPMethod(s string) string {
	switch s {
	case "get":
		return http.MethodGet
	case "post":
		return http.MethodPost
	case "put":
		return http.MethodPut
	case "patch":
		return http.MethodPatch
	case "delete":
		return http.MethodDelete
	default:
		return http.MethodGet
	}
}
