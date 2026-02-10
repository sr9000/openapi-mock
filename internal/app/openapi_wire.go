//go:build wireinject
// +build wireinject

package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/wire"

	petstoregen "openapi-mock/internal/generated/petstore"

	petstorestub "openapi-mock/internal/stubs/petstore"
)

type HTTPApp struct {
	Router          *chi.Mux
	PetstoreDefault *petstorestub.DefaultHandlers
	PetstorePets    *petstorestub.PetsHandlers
}

func providePetstoreHandlers(default_ *petstorestub.DefaultHandlers, pets *petstorestub.PetsHandlers) petstoregen.ServerInterface {
	return petstorestub.NewCompositeHandlers(default_, pets)
}

func provideHTTPRouter(petstoreHandler petstoregen.ServerInterface) *chi.Mux {
	r := chi.NewRouter()
	petstoregen.HandlerFromMux(petstoreHandler, r)
	return r
}

var HTTPProviderSet = wire.NewSet(
	petstorestub.NewDefaultHandlers,
	petstorestub.NewPetsHandlers,
	providePetstoreHandlers,
	provideHTTPRouter,
	wire.Struct(new(HTTPApp), "*"),
)

func InitializeHTTPApp(enableLogging bool) (*HTTPApp, error) {
	wire.Build(HTTPProviderSet)
	return nil, nil
}
