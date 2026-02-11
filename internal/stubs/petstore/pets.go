package petstore

import (
	"context"
	"log"
	gen "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/ctxkeys"
)

type PetsHandlers struct {
	EnableLogging bool
}

func NewPetsHandlers(enableLogging bool) *PetsHandlers {
	return &PetsHandlers{EnableLogging: enableLogging}
}

func (h *PetsHandlers) ListPets(ctx context.Context, request gen.ListPetsRequestObject) (gen.ListPetsResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] ListPets", reqID)
	}

	_ = request

	// ListPets200JSONResponse is an inline array type in server.gen.go, so we can
	// safely return an empty value.
	return gen.ListPets200JSONResponse{}, nil
}

func (h *PetsHandlers) CreatePet(ctx context.Context, request gen.CreatePetRequestObject) (gen.CreatePetResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] CreatePet", reqID)
	}

	_ = request

	// Spec declares an empty 201 response in this API.
	return gen.CreatePet201Response{}, nil
}

func (h *PetsHandlers) DeletePet(ctx context.Context, request gen.DeletePetRequestObject) (gen.DeletePetResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] DeletePet", reqID)
	}

	_ = request

	return gen.DeletePet204Response{}, nil
}

func (h *PetsHandlers) GetPetById(ctx context.Context, request gen.GetPetByIdRequestObject) (gen.GetPetByIdResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] GetPetById", reqID)
	}

	_ = request

	return gen.GetPetById200Response{}, nil
}
