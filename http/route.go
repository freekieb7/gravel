package http

type Route struct {
	Methods    []string
	Path       string
	Handler    HandlerFunc
	Middleware []MiddlewareFunc
}
