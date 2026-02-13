package middleware

import (
	"net/http"

	"openapi-mock/pkg/metrics"
)

// ErrorHandlers provides custom error handlers for StrictServer that record metrics
type ErrorHandlers struct {
	metrics *metrics.Metrics
}

// NewErrorHandlers creates new error handlers with metrics recording
func NewErrorHandlers(m *metrics.Metrics) *ErrorHandlers {
	return &ErrorHandlers{metrics: m}
}

// RequestErrorHandler handles request parsing errors (e.g., invalid JSON body)
// These result in 400 Bad Request
func (h *ErrorHandlers) RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if h.metrics != nil {
		endpoint := routeTemplateFromRequest(r)
		h.metrics.RecordHTTPError(r.Method, endpoint, http.StatusBadRequest, err.Error())
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// ResponseErrorHandler handles errors returned from handler functions
// These result in 500 Internal Server Error
func (h *ErrorHandlers) ResponseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if h.metrics != nil {
		endpoint := routeTemplateFromRequest(r)
		h.metrics.RecordHTTPError(r.Method, endpoint, http.StatusInternalServerError, err.Error())
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
