package http

import (
	"net/http"
	"slices"
	"strings"
)

type Router struct {
	Routes          []Route
	Middleware      []Middleware
	NotFoundHandler Handler
	// Add route tree for better performance
	staticRoutes map[string]map[string]Handler // path -> method -> handler
	hasWildcards bool
}

func NewRouter() Router {
	return Router{
		Routes:          make([]Route, 0),
		staticRoutes:    make(map[string]map[string]Handler),
		NotFoundHandler: NotFoundHandler,
	}
}

func (router *Router) GET(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodGet}, path, handler, middleware...)
}

func (router *Router) HEAD(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodHead}, path, handler, middleware...)
}

func (router *Router) POST(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodPost}, path, handler, middleware...)
}

func (router *Router) PUT(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodPut}, path, handler, middleware...)
}

func (router *Router) Patch(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodPatch}, path, handler, middleware...)
}

func (router *Router) DELETE(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodDelete}, path, handler, middleware...)
}

func (router *Router) CONNECT(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodConnect}, path, handler, middleware...)
}

func (router *Router) OPTIONS(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodOptions}, path, handler, middleware...)
}

func (router *Router) TRACE(path string, handler Handler, middleware ...Middleware) {
	router.Any([]string{http.MethodTrace}, path, handler, middleware...)
}

func (router *Router) Any(methods []string, path string, handler Handler, middleware ...Middleware) {
	// Apply middleware in reverse order
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	// Check if path has wildcards
	if strings.Contains(path, ":") || strings.Contains(path, "*") {
		router.hasWildcards = true
	} else {
		// Add to static route map for O(1) lookup
		if router.staticRoutes[path] == nil {
			router.staticRoutes[path] = make(map[string]Handler)
		}
		for _, method := range methods {
			router.staticRoutes[path][method] = handler
		}
	}

	router.Routes = append(router.Routes, Route{
		Methods: methods,
		Path:    path,
		Handler: handler,
	})
}

func (router *Router) Group(path string, groupFunc func(group *Router), middlewareList ...Middleware) {
	group := NewRouter()

	groupFunc(&group)

	for _, route := range group.Routes {
		route.Path = path + route.Path
		for _, middleware := range middlewareList {
			route.Handler = middleware(route.Handler)
		}

		router.Routes = append(router.Routes, route)
	}
}

func (router *Router) Handler() Handler {
	return func(req *Request, res *Response) {
		path := string(req.Path)
		method := string(req.Method)

		// Fast path: check static routes first (O(1) lookup)
		if methodMap, exists := router.staticRoutes[path]; exists {
			if handler, exists := methodMap[method]; exists {
				handler(req, res)
				return
			}
		}

		// Slower path: check wildcard routes if any exist
		if router.hasWildcards {
			for _, route := range router.Routes {
				if route.Path == path {
					if slices.Contains(route.Methods, method) {
						route.Handler(req, res)
						return
					}
				}
			}
		}

		// No route found
		router.NotFoundHandler(req, res)
	}
}
