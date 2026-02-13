package petstore

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	// Small deterministic dataset
	pets := gen.ListPets200JSONResponse{
		{Id: 1, Name: "Fluffy"},
		{Id: 2, Name: "Buddy"},
	}
	_ = request
	return pets, nil
}

func (h *PetsHandlers) CreatePet(ctx context.Context, request gen.CreatePetRequestObject) (gen.CreatePetResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] CreatePet", reqID)
	}

	if request.Body != nil {
		name := strings.TrimSpace(request.Body.Name)
		switch {
		case strings.HasPrefix(name, "error:"):
			return nil, fmt.Errorf("stub create pet error: %s", strings.TrimSpace(strings.TrimPrefix(name, "error:")))
		case strings.HasPrefix(name, "panic:"):
			panic(fmt.Sprintf("stub create pet panic: %s", strings.TrimSpace(strings.TrimPrefix(name, "panic:"))))
		}
	}

	return gen.CreatePet201Response{}, nil
}

func (h *PetsHandlers) DeletePet(ctx context.Context, request gen.DeletePetRequestObject) (gen.DeletePetResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] DeletePet petId=%d", reqID, request.PetId)
	}

	switch {
	case request.PetId == 500:
		return nil, fmt.Errorf("stub delete pet error: internal")
	case request.PetId == 999:
		panic("stub delete pet panic")
	}

	return gen.DeletePet204Response{}, nil
}

func (h *PetsHandlers) GetPetById(ctx context.Context, request gen.GetPetByIdRequestObject) (gen.GetPetByIdResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] GetPetById petId=%d", reqID, request.PetId)
	}

	// Drive errors/panics by ID to keep client simple
	switch {
	case request.PetId == 500:
		return nil, fmt.Errorf("stub get pet error: internal")
	case request.PetId == 404:
		// represent a 404 by returning an error; StrictServer will treat as unhandled error (500)
		// so for actual 404 you'd normally have a typed response. Keeping this simple for metrics.
		return nil, fmt.Errorf("stub get pet not found")
	case request.PetId == 999:
		panic("stub get pet panic")
	}

	return gen.GetPetById200Response{}, nil
}
