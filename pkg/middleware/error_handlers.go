package middleware

import (
	"net/http"

	"openapi-mock/pkg/metrics"
)

// ErrorHandlers provides custom error handlers for StrictServer that record metrics
type ErrorHandlers struct {
	metrics           *metrics.Metrics
	operationResolver OperationResolver
}

// NewErrorHandlers creates new error handlers with metrics recording
func NewErrorHandlers(m *metrics.Metrics, operationResolver OperationResolver) *ErrorHandlers {
	return &ErrorHandlers{metrics: m, operationResolver: operationResolver}
}

// RequestErrorHandler handles request parsing errors (e.g., invalid JSON body)
// These result in 400 Bad Request
func (h *ErrorHandlers) RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if h.metrics != nil {
		endpoint := routeTemplateFromRequest(r)
		operation := resolveOperationLabel(r, h.operationResolver)
		h.metrics.RecordHTTPError(r.Method, endpoint, operation, http.StatusBadRequest, "request_parse")
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// ResponseErrorHandler handles errors returned from handler functions
// These result in 500 Internal Server Error
func (h *ErrorHandlers) ResponseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if h.metrics != nil {
		endpoint := routeTemplateFromRequest(r)
		operation := resolveOperationLabel(r, h.operationResolver)
		h.metrics.RecordHTTPError(r.Method, endpoint, operation, http.StatusInternalServerError, "handler_error")
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
