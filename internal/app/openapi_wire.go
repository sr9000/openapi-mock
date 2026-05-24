//go:build wireinject
// +build wireinject

package app

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/wire"

	echogen0 "openapi-mock/internal/generated/echo"
	petstoregen1 "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/middleware"

	echostub0 "openapi-mock/internal/stubs/echo"
	petstorestub1 "openapi-mock/internal/stubs/petstore"
)

type HTTPApp struct {
	Router          *chi.Mux
	EchoEcho        *echostub0.EchoHandlers
	EchoStatus      *echostub0.StatusHandlers
	PetstoreDefault *petstorestub1.DefaultHandlers
	PetstorePets    *petstorestub1.PetsHandlers
}

func provideEchoStrictMiddlewares() []echogen0.StrictMiddlewareFunc {
	return []echogen0.StrictMiddlewareFunc{
		func(next echogen0.StrictHandlerFunc, operationID string) echogen0.StrictHandlerFunc {
			wrapped := middleware.OperationContext()(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
				return next(ctx, w, r, request)
			}, operationID)
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
				return wrapped(ctx, w, r, request)
			}
		},
	}
}

func provideEchoHandlers(echo *echostub0.EchoHandlers, status *echostub0.StatusHandlers, errHandlers *middleware.ErrorHandlers) echogen0.ServerInterface {
	strict := echostub0.NewCompositeHandlers(echo, status)
	strictMiddlewares := provideEchoStrictMiddlewares()
	return echogen0.NewStrictHandlerWithOptions(strict, strictMiddlewares, echogen0.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandlers.RequestErrorHandler,
		ResponseErrorHandlerFunc: errHandlers.ResponseErrorHandler,
	})
}

func providePetstoreStrictMiddlewares() []petstoregen1.StrictMiddlewareFunc {
	return []petstoregen1.StrictMiddlewareFunc{
		func(next petstoregen1.StrictHandlerFunc, operationID string) petstoregen1.StrictHandlerFunc {
			wrapped := middleware.OperationContext()(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
				return next(ctx, w, r, request)
			}, operationID)
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
				return wrapped(ctx, w, r, request)
			}
		},
	}
}

func providePetstoreHandlers(default_ *petstorestub1.DefaultHandlers, pets *petstorestub1.PetsHandlers, errHandlers *middleware.ErrorHandlers) petstoregen1.ServerInterface {
	strict := petstorestub1.NewCompositeHandlers(default_, pets)
	strictMiddlewares := providePetstoreStrictMiddlewares()
	return petstoregen1.NewStrictHandlerWithOptions(strict, strictMiddlewares, petstoregen1.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  errHandlers.RequestErrorHandler,
		ResponseErrorHandlerFunc: errHandlers.ResponseErrorHandler,
	})
}

func provideOperationResolver() middleware.OperationResolver {
	resolver := middleware.NewOperationResolver(map[string]string{
		"DELETE /pets/{petId}": "DeletePet",
		"GET /echo/headers":    "EchoHeaders",
		"GET /echo/{message}":  "EchoPath",
		"GET /health":          "HealthCheck",
		"GET /isfine":          "IsFine",
		"GET /pets":            "ListPets",
		"GET /pets/{petId}":    "GetPetById",
		"GET /status":          "GetStatus",
		"POST /echo":           "Echo",
		"POST /pets":           "CreatePet",
	})
	middleware.SetOperationResolver(resolver)
	return resolver
}

func provideErrorHandlers(m *metrics.Metrics, operationResolver middleware.OperationResolver) *middleware.ErrorHandlers {
	return middleware.NewErrorHandlers(m, operationResolver)
}

func provideHTTPRouter(middlewares []func(http.Handler) http.Handler, errHandlers *middleware.ErrorHandlers, echoHandler echogen0.ServerInterface, petstoreHandler petstoregen1.ServerInterface) *chi.Mux {
	r := chi.NewRouter()
	for _, mw := range middlewares {
		r.Use(mw)
	}
	echogen0.HandlerWithOptions(echoHandler, echogen0.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: errHandlers.RequestErrorHandler})
	petstoregen1.HandlerWithOptions(petstoreHandler, petstoregen1.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: errHandlers.RequestErrorHandler})
	return r
}

var HTTPProviderSet = wire.NewSet(
	echostub0.NewEchoHandlers,
	echostub0.NewStatusHandlers,
	provideEchoHandlers,
	petstorestub1.NewDefaultHandlers,
	petstorestub1.NewPetsHandlers,
	providePetstoreHandlers,
	provideErrorHandlers,
	provideOperationResolver,
	provideHTTPRouter,
	wire.Struct(new(HTTPApp), "*"),
)

func InitializeHTTPApp(middlewares []func(http.Handler) http.Handler, m *metrics.Metrics, enableLogging bool) (*HTTPApp, error) {
	wire.Build(HTTPProviderSet)
	return nil, nil
}
