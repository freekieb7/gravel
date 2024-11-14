package http

type Handler interface {
	ServeHTTP(*Request, *Response)
}

type HandlerFunc func(request *Request, response *Response)

func (f HandlerFunc) ServeHTTP(r *Request, w *Response) {
	f(r, w)
}

type Router struct {
	Path       string
	Routes     []Route
	Groups     []Router
	Middleware []MiddlewareFunc
}

func NewRouter() *Router {

	return &Router{
		Path:       "",
		Routes:     make([]Route, 0),
		Groups:     make([]Router, 0),
		Middleware: make([]MiddlewareFunc, 0),
	}
}

func (router *Router) Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"GET"}, path, handler, middleware...)
}

func (router *Router) Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"POST"}, path, handler, middleware...)
}

func (router *Router) Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"PUT"}, path, handler, middleware...)
}

func (router *Router) Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"PATCH"}, path, handler, middleware...)
}

func (router *Router) Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"DELETE"}, path, handler, middleware...)
}

func (router *Router) Option(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"OPTION"}, path, handler, middleware...)
}

func (router *Router) Any(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Routes = append(router.Routes, Route{
		Methods:    methods,
		Path:       path,
		Handler:    handler,
		Middleware: middleware,
	})
}

func (router *Router) Group(path string, groupFunc func(group *Router), middleware ...MiddlewareFunc) {
	group := NewRouter()
	group.Path = path
	group.Middleware = middleware

	groupFunc(group)

	router.Groups = append(router.Groups, *group)
}

func (router *Router) Add(middleware ...MiddlewareFunc) {
	router.Middleware = append(router.Middleware, middleware...)
}
