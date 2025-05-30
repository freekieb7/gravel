package http

import (
	"context"
	"net/http"
	"slices"

	"github.com/freekieb7/gravel/telemetry"
)

type Server struct {
	Name         string
	Router       Router
	ShutdownFunc func(context.Context) error
}

func NewServer(name string) Server {
	return Server{
		Name:         name,
		Router:       NewRouter(),
		ShutdownFunc: func(ctx context.Context) error { return nil },
	}
}

func (server *Server) ListenAndServe(ctx context.Context, addr string) error {
	// Setup opentelemetry
	otelShutdown, err := telemetry.Setup(ctx)
	if err != nil {
		return err
	}
	server.ShutdownFunc = otelShutdown

	// Setup routes
	routeTable := mergeRoutes(server.Router)

	for path, routes := range routeTable {
		// Method aware request handler
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			for _, route := range routes {
				if !slices.Contains(route.Methods, r.Method) {
					continue
				}

				route.Handler(&Request{r}, Response{w})
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		})
	}

	return http.ListenAndServe(addr, nil)
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.ShutdownFunc(ctx)
}

func mergeRoutes(router Router) map[string][]Route {
	routeTable := make(map[string][]Route, 0)

	// Add direct routes to table
	for _, route := range router.Routes {
		// Create new route with combined router and route configs
		path := router.Path + route.Path
		middlewares := append(router.Middleware, route.Middleware...)

		routeTable[path] = append(routeTable[path], Route{
			Path:       path,
			Methods:    route.Methods,
			Handler:    route.Handler,
			Middleware: middlewares,
		})
	}

	// Add indirect (sub router) routes to table
	for _, subRouter := range router.Groups {
		subRouterRouteTable := mergeRoutes(subRouter)

		for _, routes := range subRouterRouteTable {
			for _, route := range routes {
				// Create new route with combined router and route configs
				path := router.Path + route.Path
				middleware := append(router.Middleware, route.Middleware...)

				routeTable[path] = append(routeTable[path], Route{
					Path:       path,
					Methods:    route.Methods,
					Handler:    route.Handler,
					Middleware: middleware,
				})
			}
		}
	}

	return routeTable
}
