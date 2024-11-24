package http

type Handler interface {
	ServeHTTP(Request, Response)
}

type HandlerFunc func(request Request, response Response)

func (handlerFunc HandlerFunc) ServeHTTP(request Request, response Response) {
	handlerFunc(request, response)
}

type Router interface {
	Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Connect(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Options(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Trace(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc)
	Any(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc)

	Group(path string, groupFunc func(group Router), middleware ...MiddlewareFunc)

	Middleware() []MiddlewareFunc
	AddMiddleware(middleware ...MiddlewareFunc)
	SetMiddleware(middleware ...MiddlewareFunc)

	Path() string
	SetPath(path string)

	Routes() []Route
	Groups() []Router
}

type router struct {
	path       string
	routes     []Route
	groups     []Router
	middleware []MiddlewareFunc
}

func NewRouter() Router {
	return &router{
		path:       "",
		routes:     make([]Route, 0),
		groups:     make([]Router, 0),
		middleware: make([]MiddlewareFunc, 0),
	}
}

func (router *router) Get(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"GET"}, path, handler, middleware...)
}

func (router *router) Post(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"POST"}, path, handler, middleware...)
}

func (router *router) Put(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"PUT"}, path, handler, middleware...)
}

func (router *router) Delete(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"DELETE"}, path, handler, middleware...)
}

func (router *router) Connect(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"CONNECT"}, path, handler, middleware...)
}

func (router *router) Options(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"OPTIONS"}, path, handler, middleware...)
}

func (router *router) Trace(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"TRACE"}, path, handler, middleware...)
}

func (router *router) Option(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"OPTION"}, path, handler, middleware...)
}

func (router *router) Patch(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.Any([]string{"PATCH"}, path, handler, middleware...)
}

func (router *router) Any(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	router.routes = append(router.routes, Route{
		Methods:    methods,
		Path:       path,
		Handler:    handler,
		Middleware: middleware,
	})
}

func (router *router) Middleware() []MiddlewareFunc {
	return router.middleware
}

func (router *router) AddMiddleware(middleware ...MiddlewareFunc) {
	router.middleware = append(router.middleware, middleware...)
}

func (router *router) SetMiddleware(middleware ...MiddlewareFunc) {
	router.middleware = middleware
}

func (router *router) Path() string {
	return router.path
}

func (router *router) SetPath(path string) {
	router.path = path
}

func (router *router) Group(path string, groupFunc func(group Router), middleware ...MiddlewareFunc) {
	group := NewRouter()
	group.SetPath(path)
	group.SetMiddleware(middleware...)

	groupFunc(group)

	router.groups = append(router.groups, group)
}

func (router *router) Routes() []Route {
	return router.routes
}

func (router *router) Groups() []Router {
	return router.groups
}
