package petstore

import (
	"context"
	"log"
	gen "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/ctxkeys"
)

type DefaultHandlers struct {
	EnableLogging bool
}

func NewDefaultHandlers(enableLogging bool) *DefaultHandlers {
	return &DefaultHandlers{EnableLogging: enableLogging}
}

func (h *DefaultHandlers) HealthCheck(ctx context.Context, request gen.HealthCheckRequestObject) (gen.HealthCheckResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [DefaultHandlers] HealthCheck", reqID)
	}

	_ = request

	return gen.HealthCheck200Response{}, nil
}
