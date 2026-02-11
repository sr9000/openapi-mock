package echo

import (
	"log"
	"net/http"
	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/ctxkeys"

	"github.com/labstack/echo/v4"
)

type EchoHandlers struct {
	EnableLogging bool
}

func NewEchoHandlers(enableLogging bool) *EchoHandlers {
	return &EchoHandlers{EnableLogging: enableLogging}
}

func (h *EchoHandlers) EchoHeaders(ctx echo.Context) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] EchoHeaders", reqID)
	}

	return ctx.JSON(http.StatusOK, gen.HeadersResponse{})
}

func (h *EchoHandlers) EchoPath(ctx echo.Context, message string) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] EchoPath", reqID)
	}

	return ctx.JSON(http.StatusOK, gen.EchoResponse{})
}

func (h *EchoHandlers) Echo(ctx echo.Context) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [EchoHandlers] Echo", reqID)
	}

	return ctx.JSON(http.StatusOK, gen.EchoResponse{})
}
