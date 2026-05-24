package echo

import (
	"context"

	"github.com/rs/zerolog"

	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/observability"
)

type EchoHandlers struct {
	EnableLogging bool
}

func NewEchoHandlers(enableLogging bool) *EchoHandlers {
	return &EchoHandlers{EnableLogging: enableLogging}
}

func (h *EchoHandlers) logger(ctx context.Context) zerolog.Logger {
	return observability.Logger(ctx, zerolog.Nop())
}

func (h *EchoHandlers) Echo(ctx context.Context, request gen.EchoRequestObject) (gen.EchoResponseObject, error) {
	if h.EnableLogging {
		logger := h.logger(ctx)
		logger.Info().Str("handler", "EchoHandlers").Msg("Echo")
	}

	echoValue := ""
	if request.JSONBody != nil && request.JSONBody.Message != nil {
		echoValue = *request.JSONBody.Message
	} else if request.TextBody != nil {
		echoValue = string(*request.TextBody)
	}

	return gen.Echo200JSONResponse{Echo: echoValue}, nil
}

func (h *EchoHandlers) EchoHeaders(ctx context.Context, request gen.EchoHeadersRequestObject) (gen.EchoHeadersResponseObject, error) {
	if h.EnableLogging {
		logger := h.logger(ctx)
		logger.Info().Str("handler", "EchoHandlers").Msg("EchoHeaders")
	}

	_ = request
	headers := map[string]string{}
	if requestID := observability.RequestID(ctx); requestID != "" {
		headers["X-Request-ID"] = requestID
	}
	if traceID := observability.TraceID(ctx); traceID != "" {
		headers["X-Trace-ID"] = traceID
	}

	return gen.EchoHeaders200JSONResponse{Headers: &headers}, nil
}

func (h *EchoHandlers) EchoPath(ctx context.Context, request gen.EchoPathRequestObject) (gen.EchoPathResponseObject, error) {
	if h.EnableLogging {
		logger := h.logger(ctx)
		logger.Info().Str("handler", "EchoHandlers").Msg("EchoPath")
	}

	return gen.EchoPath200JSONResponse{Echo: request.Message}, nil
}
