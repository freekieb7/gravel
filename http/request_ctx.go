package http

import (
	"bufio"
	"net"
)

type RequestCtx struct {
	Conn       net.Conn
	ConnReader bufio.Reader
	ConnWriter bufio.Writer

	Request  Request
	Response Response
}

func (reqCtx *RequestCtx) Reset(conn net.Conn) {
	reqCtx.Conn = conn
	reqCtx.ConnReader.Reset(conn)
	reqCtx.ConnWriter.Reset(conn)
}
