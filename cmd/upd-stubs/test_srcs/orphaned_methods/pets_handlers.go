//go:build ignore

package petstore

type PetsHandlers struct{}

func (h *PetsHandlers) ListPets(ctx context.Context, request any) (any, error) {
	return nil, nil
}

func (h *PetsHandlers) DeprecatedPets(ctx context.Context, request any) (any, error) {
	return nil, nil
}
