package echo

import (
	"context"
	"log"
	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/ctxkeys"
)

type EchoHandlers struct {
	EnableLogging bool
}

func NewEchoHandlers(enableLogging bool) *EchoHandlers {
	return &EchoHandlers{EnableLogging: enableLogging}
}

func (h *EchoHandlers) EchoHeaders(ctx context.Context, request gen.EchoHeadersRequestObject) (gen.EchoHeadersResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] EchoHeaders", reqID)
	}

	_ = request

	return gen.EchoHeaders200JSONResponse(gen.HeadersResponse{}), nil
}

func (h *EchoHandlers) EchoPath(ctx context.Context, request gen.EchoPathRequestObject) (gen.EchoPathResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] EchoPath", reqID)
	}

	_ = request

	return gen.EchoPath200JSONResponse(gen.EchoResponse{}), nil
}

func (h *EchoHandlers) Echo(ctx context.Context, request gen.EchoRequestObject) (gen.EchoResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] Echo", reqID)
	}

	_ = request

	return gen.Echo200JSONResponse(gen.EchoResponse{}), nil
}
