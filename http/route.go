package http

type Route struct {
	Methods        []string
	Path           string
	HandleFunc     HandleFunc
	MiddlewareFunc []MiddlewareFunc
}

var NotFoundHandleFunc HandleFunc = func(ctx *RequestCtx) {
	ctx.Response.WithStatus(StatusNotFound)
}
