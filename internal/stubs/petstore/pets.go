package petstore

import (
	"log"
	"net/http"
	gen "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/ctxkeys"

	"github.com/labstack/echo/v4"
)

type PetsHandlers struct {
	EnableLogging bool
}

func NewPetsHandlers(enableLogging bool) *PetsHandlers {
	return &PetsHandlers{EnableLogging: enableLogging}
}

func (h *PetsHandlers) ListPets(ctx echo.Context, params gen.ListPetsParams) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] ListPets", reqID)
	}

	// Return a basic mock payload (spec doesn't currently generate Pet schema types)
	pets := []map[string]any{
		{"id": 1, "name": "Fluffy"},
		{"id": 2, "name": "Buddy"},
		{"id": 3, "name": "Max"},
	}
	return ctx.JSON(http.StatusOK, pets)
}

func (h *PetsHandlers) CreatePet(ctx echo.Context) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] CreatePet", reqID)
	}

	var body gen.CreatePetJSONBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return ctx.JSON(http.StatusCreated, map[string]any{"id": 123, "name": body.Name, "tag": body.Tag})
}

func (h *PetsHandlers) DeletePet(ctx echo.Context, petId int64) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] DeletePet petId=%d", reqID, petId)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *PetsHandlers) GetPetById(ctx echo.Context, petId int64) error {
	if h.EnableLogging {
		reqID, _ := ctx.Request().Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] GetPetById petId=%d", reqID, petId)
	}

	return ctx.JSON(http.StatusOK, map[string]any{"id": petId, "name": "Mock Pet"})
}
