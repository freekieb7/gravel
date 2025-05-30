package http

import "net/http"

type Handler interface {
	ServeHTTP(*Request, Response)
}

type HandleFunc func(request *Request, response Response)

func (handleFunc HandleFunc) ServeHTTP(request *Request, response Response) {
	handleFunc(request, response)
}

type Router struct {
	Path       string
	Routes     []Route
	Groups     []Router
	Middleware []MiddlewareFunc
}

func NewRouter() Router {
	return Router{
		Path:       "",
		Routes:     make([]Route, 0),
		Groups:     make([]Router, 0),
		Middleware: make([]MiddlewareFunc, 0),
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
	router.Routes = append(router.Routes, Route{
		Methods:    methods,
		Path:       path,
		Handler:    handleFunc,
		Middleware: middlewareFunc,
	})
}

func (router *Router) Group(path string, groupFunc func(group Router), middleware ...MiddlewareFunc) {
	group := NewRouter()
	group.Path = path

	groupFunc(group)

	router.Groups = append(router.Groups, group)
}
