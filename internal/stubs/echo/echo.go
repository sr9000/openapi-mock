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

func (h *EchoHandlers) Echo(ctx context.Context, request gen.EchoRequestObject) (gen.EchoResponseObject, error) {
	if h.EnableLogging {
		logger := observability.Logger(ctx, zerolog.Nop())
		logger.Info().Str("handler", "EchoHandlers").Msg("Echo")
	}

	_ = request

	return gen.Echo200JSONResponse{}, nil
}

func (h *EchoHandlers) EchoHeaders(ctx context.Context, request gen.EchoHeadersRequestObject) (gen.EchoHeadersResponseObject, error) {
	if h.EnableLogging {
		logger := observability.Logger(ctx, zerolog.Nop())
		logger.Info().Str("handler", "EchoHandlers").Msg("EchoHeaders")
	}

	_ = request

	return gen.EchoHeaders200JSONResponse{}, nil
}

func (h *EchoHandlers) EchoPath(ctx context.Context, request gen.EchoPathRequestObject) (gen.EchoPathResponseObject, error) {
	if h.EnableLogging {
		logger := observability.Logger(ctx, zerolog.Nop())
		logger.Info().Str("handler", "EchoHandlers").Msg("EchoPath")
	}

	_ = request

	return gen.EchoPath200JSONResponse{}, nil
}
