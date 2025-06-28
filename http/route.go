package http

type Route struct {
	Methods []string
	Path    string
	Handler Handler
}

var NotFoundHandler Handler = func(ctx *RequestCtx) {
	ctx.Response.WithStatus(StatusNotFound)
}
