//go:build ignore

package petstore

import "context"

type PetsHandlers struct{}

func (handler *PetsHandlers) ListPets(
	ctx context.Context,
	request any,
) (any, error) {
	return nil, nil
}

func (h PetsHandlers) GetPet(ctx context.Context, request any) (any, error) {
	return nil, nil
}
