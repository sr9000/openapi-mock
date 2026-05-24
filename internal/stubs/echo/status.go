package echo

import (
	"context"

	"github.com/rs/zerolog"

	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/observability"
)

type StatusHandlers struct {
	EnableLogging bool
}

func NewStatusHandlers(enableLogging bool) *StatusHandlers {
	return &StatusHandlers{EnableLogging: enableLogging}
}

func (h *StatusHandlers) IsFine(ctx context.Context, request gen.IsFineRequestObject) (gen.IsFineResponseObject, error) {
	if h.EnableLogging {
		logger := observability.Logger(ctx, zerolog.Nop())
		logger.Info().Str("handler", "StatusHandlers").Msg("IsFine")
	}

	_ = request

	return gen.IsFine218JSONResponse{}, nil
}

func (h *StatusHandlers) GetStatus(ctx context.Context, request gen.GetStatusRequestObject) (gen.GetStatusResponseObject, error) {
	if h.EnableLogging {
		logger := observability.Logger(ctx, zerolog.Nop())
		logger.Info().Str("handler", "StatusHandlers").Msg("GetStatus")
	}

	_ = request

	return gen.GetStatus200JSONResponse{}, nil
}
