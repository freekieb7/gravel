package http

import "net/http"

type Server interface {
	Router() Router
	SetRouter(router Router)

	Listen() error
}

type server struct {
	router Router
}

func NewServer() Server {
	return &server{
		router: NewRouter(),
	}
}

func (server *server) Router() Router {
	return server.router
}

func (server *server) SetRouter(router Router) {
	server.router = router
}

func (server *server) Listen() error {
	server.merge("", server.router)

	return http.ListenAndServe(":8080", nil)
}

func (server *server) merge(basePath string, parentGroup Router) {
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
		server.merge(basePath+parentGroup.Path(), childGroup)
	}
}
