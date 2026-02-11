package echo

import (
	"encoding/json"
	"log"
	"net/http"
	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/ctxkeys"
)

type EchoHandlers struct {
	EnableLogging bool
}

func NewEchoHandlers(enableLogging bool) *EchoHandlers {
	return &EchoHandlers{EnableLogging: enableLogging}
}

func (h *EchoHandlers) EchoHeaders(w http.ResponseWriter, r *http.Request) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] EchoHeaders", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gen.HeadersResponse{})
}

func (h *EchoHandlers) EchoPath(w http.ResponseWriter, r *http.Request, message string) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] EchoPath", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gen.EchoResponse{})
}

func (h *EchoHandlers) Echo(w http.ResponseWriter, r *http.Request) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] Echo", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gen.EchoResponse{})
}
