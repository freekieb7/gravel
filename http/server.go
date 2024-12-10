package http

import (
	"context"
	"net/http"

	"github.com/freekieb7/gravel/telemetry"
	"github.com/valyala/fasthttp"
)

type Server interface {
	Router() Router
	SetRouter(router Router)

	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type server struct {
	name        string
	router      Router
	otelShudown func(context.Context) error
}

func NewServer(name string) Server {
	return &server{
		name:        name,
		router:      NewRouter(),
		otelShudown: func(ctx context.Context) error { return nil },
	}
}

func (server *server) Router() Router {
	return server.router
}

func (server *server) SetRouter(router Router) {
	server.router = router
}

func (server *server) Run(ctx context.Context) error {
	// Setup opentelemetry
	otelShutdown, err := telemetry.Setup(ctx)
	if err != nil {
		return err
	}
	server.otelShudown = otelShutdown

	// Setup routes
	server.buildRoutes("", server.router)

	return fasthttp.ListenAndServe(":8080", nil)
}

func (server *server) Shutdown(ctx context.Context) error {
	return server.otelShudown(ctx)
}

func (server *server) buildRoutes(basePath string, parentGroup Router) {
	for _, route := range parentGroup.Routes() {
		path := basePath + parentGroup.Path() + route.Path

		routeWithMiddleware := MethodCheckMiddleware(route.Methods, RecoverMiddleware(route.Handler))
		for _, middleware := range append(parentGroup.Middleware(), route.Middleware...) {
			routeWithMiddleware = middleware(routeWithMiddleware)
		}

		// Serve HTTP
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			routeWithMiddleware.ServeHTTP(&request{r}, &response{w})
		})
	}

	// Process the branching endpoints
	for _, childGroup := range parentGroup.Groups() {
		childGroup.SetMiddleware(append(parentGroup.Middleware(), childGroup.Middleware()...)...)
		server.buildRoutes(basePath+parentGroup.Path(), childGroup)
	}
}
