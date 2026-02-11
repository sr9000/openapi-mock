//go:build wireinject
// +build wireinject

package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/wire"

	echogen "openapi-mock/internal/generated/echo"
	petstoregen "openapi-mock/internal/generated/petstore"

	echostub "openapi-mock/internal/stubs/echo"
	petstorestub "openapi-mock/internal/stubs/petstore"
)

type HTTPApp struct {
	Router          *chi.Mux
	EchoEcho        *echostub.EchoHandlers
	EchoStatus      *echostub.StatusHandlers
	PetstoreDefault *petstorestub.DefaultHandlers
	PetstorePets    *petstorestub.PetsHandlers
}

func provideEchoHandlers(echo *echostub.EchoHandlers, status *echostub.StatusHandlers) echogen.ServerInterface {
	return echostub.NewCompositeHandlers(echo, status)
}

func providePetstoreHandlers(default_ *petstorestub.DefaultHandlers, pets *petstorestub.PetsHandlers) petstoregen.ServerInterface {
	return petstorestub.NewCompositeHandlers(default_, pets)
}

func provideHTTPRouter(middlewares []func(http.Handler) http.Handler, echoHandler echogen.ServerInterface, petstoreHandler petstoregen.ServerInterface) *chi.Mux {
	r := chi.NewRouter()
	for _, mw := range middlewares {
		r.Use(mw)
	}
	echogen.HandlerFromMux(echoHandler, r)
	petstoregen.HandlerFromMux(petstoreHandler, r)
	return r
}

var HTTPProviderSet = wire.NewSet(
	echostub.NewEchoHandlers,
	echostub.NewStatusHandlers,
	provideEchoHandlers,
	petstorestub.NewDefaultHandlers,
	petstorestub.NewPetsHandlers,
	providePetstoreHandlers,
	provideHTTPRouter,
	wire.Struct(new(HTTPApp), "*"),
)

func InitializeHTTPApp(middlewares []func(http.Handler) http.Handler, enableLogging bool) (*HTTPApp, error) {
	wire.Build(HTTPProviderSet)
	return nil, nil
}
