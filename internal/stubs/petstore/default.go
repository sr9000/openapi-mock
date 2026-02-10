package petstore

import (
	"encoding/json"
	"log"
	"net/http"
	"openapi-mock/pkg/ctxkeys"
)

type DefaultHandlers struct {
	EnableLogging bool
}

func NewDefaultHandlers(enableLogging bool) *DefaultHandlers {
	return &DefaultHandlers{EnableLogging: enableLogging}
}

func (h *DefaultHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [DefaultHandlers] HealthCheck", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
