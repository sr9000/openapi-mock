package echo

import (
	"encoding/json"
	"log"
	"net/http"
	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/ctxkeys"
)

type StatusHandlers struct {
	EnableLogging bool
}

func NewStatusHandlers(enableLogging bool) *StatusHandlers {
	return &StatusHandlers{EnableLogging: enableLogging}
}

func (h *StatusHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [StatusHandlers] GetStatus", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gen.StatusResponse{})
}
