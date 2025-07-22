package http

type Route struct {
	Methods []string
	Path    string
	Handler Handler
}

var NotFoundHandler Handler = func(req *Request, res *Response) {
	res.Status = StatusNotFound
}
