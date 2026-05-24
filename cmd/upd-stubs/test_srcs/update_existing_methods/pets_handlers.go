//go:build ignore

package petstore

import (
	"context"

	gen "openapi-mock/internal/generated/petstore"
)

type PetsHandlers struct {
	EnableLogging bool
}

func NewPetsHandlers(enableLogging bool) *PetsHandlers {
	return &PetsHandlers{EnableLogging: enableLogging}
}

func (handler *PetsHandlers) ListPets(
	ctx context.Context,
	request gen.ListPetsRequestObject,
) (gen.ListPetsResponseObject, error) {
	_ = ctx
	_ = request
	return gen.ListPets200JSONResponse{}, nil
}
