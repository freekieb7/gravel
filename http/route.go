package http

type Route struct {
	Methods    []string
	Path       string
	Handler    HandleFunc
	Middleware []MiddlewareFunc
}
