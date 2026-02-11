package petstore

import (
	"log"
	"net/http"
	"openapi-mock/pkg/ctxkeys"

	"github.com/labstack/echo/v4"
)

type DefaultHandlers struct {
	EnableLogging bool
}

func NewDefaultHandlers(enableLogging bool) *DefaultHandlers {
	return &DefaultHandlers{EnableLogging: enableLogging}
}

func (h *DefaultHandlers) HealthCheck(ctx echo.Context) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [DefaultHandlers] HealthCheck", reqID)
	}

	return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
