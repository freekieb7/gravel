package http

import (
	"net"
)

type RequestCtx struct {
	Conn net.Conn

	Request  Request
	Response Response
}

func (ctx *RequestCtx) Reset() {
	ctx.Request.Reset()
	ctx.Response.Reset()
	ctx.Conn = nil
}
