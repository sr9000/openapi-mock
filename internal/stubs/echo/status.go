package echo

import (
	"context"
	"log"
	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/ctxkeys"
)

type StatusHandlers struct {
	EnableLogging bool
}

func NewStatusHandlers(enableLogging bool) *StatusHandlers {
	return &StatusHandlers{EnableLogging: enableLogging}
}

func (h *StatusHandlers) IsFine(ctx context.Context, request gen.IsFineRequestObject) (gen.IsFineResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [StatusHandlers] IsFine", reqID)
	}

	_ = request

	return gen.IsFine218JSONResponse{}, nil
}

func (h *StatusHandlers) GetStatus(ctx context.Context, request gen.GetStatusRequestObject) (gen.GetStatusResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [StatusHandlers] GetStatus", reqID)
	}

	_ = request

	return gen.GetStatus200JSONResponse{}, nil
}
