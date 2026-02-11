//go:build wireinject
// +build wireinject

package app

import (
	"net/http"

	"github.com/google/wire"
	"github.com/labstack/echo/v4"

	echogen "openapi-mock/internal/generated/echo"
	petstoregen "openapi-mock/internal/generated/petstore"

	echostub "openapi-mock/internal/stubs/echo"
	petstorestub "openapi-mock/internal/stubs/petstore"
)

type HTTPApp struct {
	Router          *echo.Echo
	EchoEcho        *echostub.EchoHandlers
	EchoStatus      *echostub.StatusHandlers
	PetstoreDefault *petstorestub.DefaultHandlers
	PetstorePets    *petstorestub.PetsHandlers
}

func provideEchoHandlers(echoH *echostub.EchoHandlers, status *echostub.StatusHandlers) echogen.ServerInterface {
	return echostub.NewCompositeHandlers(echoH, status)
}

func providePetstoreHandlers(default_ *petstorestub.DefaultHandlers, pets *petstorestub.PetsHandlers) petstoregen.ServerInterface {
	return petstorestub.NewCompositeHandlers(default_, pets)
}

func provideHTTPRouter(middlewares []func(http.Handler) http.Handler, echoHandler echogen.ServerInterface, petstoreHandler petstoregen.ServerInterface) *echo.Echo {
	e := echo.New()

	// Adapt net/http middleware to Echo
	for _, mw := range middlewares {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					c.SetRequest(r)
					err := next(c)
					if err != nil {
						_ = c.Error(err)
					}
				}))
				h.ServeHTTP(c.Response(), c.Request())
				return nil
			}
		})
	}

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
	provideHTTPRouter,
	wire.Struct(new(HTTPApp), "*"),
)

func InitializeHTTPApp(middlewares []func(http.Handler) http.Handler, enableLogging bool) (*HTTPApp, error) {
	wire.Build(HTTPProviderSet)
	return nil, nil
}
