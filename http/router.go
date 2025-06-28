package http

import "net/http"

type Handler func(ctx *RequestCtx)

type Router struct {
	Routes     []Route
	Middleware []Middleware
}

func NewRouter() Router {
	return Router{
		Routes: make([]Route, 0),
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
	for _, middleware := range middleware {
		handler = middleware(handler)
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
	return func(ctx *RequestCtx) {
		handler := NotFoundHandler
		for _, route := range router.Routes {
			if route.Path != string(ctx.Request.Path) {
				continue
			}

			for _, method := range route.Methods {
				if method != string(ctx.Request.Method) {
					continue
				}

				handler = route.Handler
				break
			}
		}

		handler(ctx)
	}
}
