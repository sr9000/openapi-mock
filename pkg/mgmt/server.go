package mgmt

import (
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/recorder"
)

//go:embed openapi.json
var openapiSpec []byte

//go:embed swagger-ui-bundle.js
var swaggerUIBundleJS []byte

//go:embed swagger-ui.css
var swaggerUICSS []byte

// Swagger UI HTML template that loads embedded assets
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>OpenAPI Mock Management API</title>
    <link rel="stylesheet" href="/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "/openapi.json",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout"
            });
        };
    </script>
</body>
</html>`

// Server is the management HTTP server for e2e testing.
type Server struct {
	recorder      *recorder.Recorder
	contextValues *mm.Store
	server        *http.Server
	port          string
}

// New creates a new management server
func New(rec *recorder.Recorder, contextValues *mm.Store, port string) *Server {
	if contextValues == nil {
		contextValues = mm.NewStore()
	}
	return &Server{
		recorder:      rec,
		contextValues: contextValues,
		port:          port,
	}
}

// Start starts the management HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: s.router(),
	}

	log.Printf("starting management server on port %s", s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("management server error: %v", err)
		}
	}()

	return nil
}

func (s *Server) router() http.Handler {
	r := chi.NewRouter()
	r.Get("/logs", s.handleLogs)
	r.Delete("/logs", s.handleDeleteLogs)
	r.Get("/logs/{request_id}", s.handleLogsByRequestID)

	r.Get("/context-values", s.handleGetContextValues)
	r.Put("/context-values", s.handlePutContextValues)
	r.Patch("/context-values", s.handlePatchContextValues)
	r.Delete("/context-values", s.handleDeleteContextValues)

	r.Get("/context-values/{request_id}", s.handleGetContextValuesByRequestID)
	r.Put("/context-values/{request_id}", s.handlePutContextValuesByRequestID)
	r.Patch("/context-values/{request_id}", s.handlePatchContextValuesByRequestID)
	r.Delete("/context-values/{request_id}", s.handleDeleteContextValuesByRequestID)

	r.Get("/doc", s.handleDoc)
	r.Get("/openapi.json", s.handleOpenAPI)
	r.Get("/swagger-ui-bundle.js", s.handleSwaggerUIBundle)
	r.Get("/swagger-ui.css", s.handleSwaggerUICSS)
	return r
}

// Stop gracefully stops the management server
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleLogs returns all recorded HTTP/OpenAPI calls as JSON.
func (s *Server) handleLogs(w http.ResponseWriter, _ *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	data, err := s.recorder.ToJSON()
	if err != nil {
		http.Error(w, "failed to serialize logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// handleLogsByRequestID returns all records for request id from /logs/{request_id}.
func (s *Server) handleLogsByRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "request_id")

	data, err := json.Marshal(s.recorder.GetRecordsByRequestID(requestID))
	if err != nil {
		http.Error(w, "failed to serialize logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) handleDeleteLogs(w http.ResponseWriter, _ *http.Request) {
	s.recorder.Clear()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleGetContextValues(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.contextValues.GetAll())
}

func (s *Server) handleGetContextValuesByRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "request_id")
	values := s.contextValues.Get(requestID)
	if values == nil {
		values = map[string]any{}
	}
	writeJSON(w, http.StatusOK, values)
}

func (s *Server) handlePutContextValues(w http.ResponseWriter, r *http.Request) {
	data, ok := s.decodeStoreBody(w, r)
	if !ok {
		return
	}
	s.contextValues.ReplaceAll(data)
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handlePutContextValuesByRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "request_id")
	data, ok := s.decodeObjectBody(w, r)
	if !ok {
		return
	}
	s.contextValues.Replace(requestID, data)
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handlePatchContextValues(w http.ResponseWriter, r *http.Request) {
	data, ok := s.decodeStoreBody(w, r)
	if !ok {
		return
	}
	s.contextValues.MergeAll(data)
	writeJSON(w, http.StatusOK, s.contextValues.GetAll())
}

func (s *Server) handlePatchContextValuesByRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "request_id")
	data, ok := s.decodeObjectBody(w, r)
	if !ok {
		return
	}
	s.contextValues.Merge(requestID, data)
	writeJSON(w, http.StatusOK, s.contextValues.Get(requestID))
}

func (s *Server) handleDeleteContextValues(w http.ResponseWriter, _ *http.Request) {
	s.contextValues.Clear()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleDeleteContextValuesByRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "request_id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) == 0 {
		s.contextValues.Delete(requestID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}
	var req struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Keys) == 0 {
		s.contextValues.Delete(requestID)
	} else {
		s.contextValues.DeleteKeys(requestID, req.Keys)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleDoc serves the interactive Swagger UI page
func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerUIHTML))
}

// handleOpenAPI returns the OpenAPI specification JSON
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(openapiSpec)
}

// handleSwaggerUIBundle serves the embedded swagger-ui-bundle.js
func (s *Server) handleSwaggerUIBundle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(swaggerUIBundleJS)
}

// handleSwaggerUICSS serves the embedded swagger-ui.css
func (s *Server) handleSwaggerUICSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	w.Write(swaggerUICSS)
}

func (s *Server) decodeObjectBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return nil, false
	}
	data, err := mm.DecodeObject(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	return data, true
}

func (s *Server) decodeStoreBody(w http.ResponseWriter, r *http.Request) (map[string]map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return nil, false
	}
	data, err := mm.DecodeStore(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	return data, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to serialize response", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
