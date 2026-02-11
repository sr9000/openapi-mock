//go:build wireinject
// +build wireinject

package app

import (
	"net/http"

	"github.com/google/wire"
	"github.com/labstack/echo/v4"
	mw "github.com/labstack/echo/v4/middleware"

	echogen "openapi-mock/internal/generated/echo"
	petstoregen "openapi-mock/internal/generated/petstore"

	echostub "openapi-mock/internal/stubs/echo"
	petstorestub "openapi-mock/internal/stubs/petstore"
)

type HTTPApp struct {
	Echo            *echo.Echo
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

func provideHTTPEcho(middlewares []func(http.Handler) http.Handler, echoHandler echogen.ServerInterface, petstoreHandler petstoregen.ServerInterface) *echo.Echo {
	e := echo.New()
	for _, mwHTTP := range middlewares {
		e.Use(echo.WrapMiddleware(mwHTTP))
	}
	e.HideBanner = true
	e.Use(mw.Recover())
	echogen.RegisterHandlers(e, echoHandler)
	petstoregen.RegisterHandlers(e, petstoreHandler)
	return e
}

var HTTPProviderSet = wire.NewSet(
	echostub.NewEchoHandlers,
	echostub.NewStatusHandlers,
	provideEchoHandlers,
	petstorestub.NewDefaultHandlers,
	petstorestub.NewPetsHandlers,
	providePetstoreHandlers,
	provideHTTPEcho,
	wire.Struct(new(HTTPApp), "*"),
)

func InitializeHTTPApp(middlewares []func(http.Handler) http.Handler, enableLogging bool) (*HTTPApp, error) {
	wire.Build(HTTPProviderSet)
	return nil, nil
}
