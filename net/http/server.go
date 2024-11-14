package http

import (
	"net/http"
)

type Server struct {
	Router *Router
}

func NewServer() *Server {
	return &Server{
		Router: NewRouter(),
	}
}

func (server *Server) Listen() {
	server.merge("", *server.Router)

	http.ListenAndServe(":8080", nil)
}

func (server *Server) merge(basePath string, group Router) {
	for _, route := range group.Routes {
		path := basePath + group.Path + route.Path

		routeWithMiddleware := MethodCheckMiddleware(route.Methods, RecoverMiddleware(route.Handler))
		for _, middleware := range append(group.Middleware, route.Middleware...) {
			routeWithMiddleware = middleware(routeWithMiddleware)
		}

		// Serve HTTP
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			routeWithMiddleware.ServeHTTP(&Request{r}, &Response{w})
		})
	}

	// Process the branching endpoints
	for _, subGroup := range group.Groups {
		subGroup.Middleware = append(group.Middleware, subGroup.Middleware...)
		server.merge(basePath+group.Path, subGroup)
	}

}
