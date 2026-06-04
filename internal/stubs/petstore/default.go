package petstore

import (
	"context"

	"github.com/rs/zerolog"

	gen "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/observability"
)

type DefaultHandlers struct {
	EnableLogging bool
}

func NewDefaultHandlers(enableLogging bool) *DefaultHandlers {
	return &DefaultHandlers{EnableLogging: enableLogging}
}

func (h *DefaultHandlers) logger(ctx context.Context) zerolog.Logger {
	return observability.Logger(ctx, zerolog.Nop())
}

func (h *DefaultHandlers) HealthCheck(ctx context.Context, request gen.HealthCheckRequestObject) (gen.HealthCheckResponseObject, error) {
	if h.EnableLogging {
		logger := h.logger(ctx)
		logger.Info().Str("handler", "DefaultHandlers").Msg("HealthCheck")
	}

	_ = request

	return gen.HealthCheck200Response{}, nil
}
