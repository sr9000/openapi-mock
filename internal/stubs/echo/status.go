package echo

import (
	"log"
	"net/http"
	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/ctxkeys"

	"github.com/labstack/echo/v4"
)

type StatusHandlers struct {
	EnableLogging bool
}

func NewStatusHandlers(enableLogging bool) *StatusHandlers {
	return &StatusHandlers{EnableLogging: enableLogging}
}

func (h *StatusHandlers) GetStatus(ctx echo.Context) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [StatusHandlers] GetStatus", reqID)
	}

	return ctx.JSON(http.StatusOK, gen.StatusResponse{})
}
