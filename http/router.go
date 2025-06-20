package http

import "net/http"

type Handler interface {
	ServeHTTP(*RequestCtx)
}

type HandleFunc func(ctx *RequestCtx)

type Router struct {
	Routes     []Route
	Middleware []MiddlewareFunc
}

func NewRouter() Router {
	return Router{
		Routes: make([]Route, 0),
	}
}

func (router *Router) GET(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodGet}, path, handleFunc, middlewareFunc...)
}

func (router *Router) HEAD(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodHead}, path, handleFunc, middlewareFunc...)
}

func (router *Router) POST(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodPost}, path, handleFunc, middlewareFunc...)
}

func (router *Router) PUT(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodPut}, path, handleFunc, middlewareFunc...)
}

func (router *Router) Patch(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodPatch}, path, handleFunc, middlewareFunc...)
}

func (router *Router) DELETE(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodDelete}, path, handleFunc, middlewareFunc...)
}

func (router *Router) CONNECT(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodConnect}, path, handleFunc, middlewareFunc...)
}

func (router *Router) OPTIONS(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodOptions}, path, handleFunc, middlewareFunc...)
}

func (router *Router) TRACE(path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	router.Any([]string{http.MethodTrace}, path, handleFunc, middlewareFunc...)
}

func (router *Router) Any(methods []string, path string, handleFunc HandleFunc, middlewareFunc ...MiddlewareFunc) {
	for _, middleware := range middlewareFunc {
		handleFunc = middleware(handleFunc)
	}

	router.Routes = append(router.Routes, Route{
		Methods:    methods,
		Path:       path,
		HandleFunc: handleFunc,
	})
}

func (router *Router) Group(path string, groupFunc func(group *Router), middlewareFunc ...MiddlewareFunc) {
	group := NewRouter()

	groupFunc(&group)

	for _, route := range group.Routes {
		route.Path = path + route.Path
		for _, middleware := range middlewareFunc {
			route.HandleFunc = middleware(route.HandleFunc)
		}

		router.Routes = append(router.Routes, route)
	}
}
