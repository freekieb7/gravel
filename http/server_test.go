package http

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"testing"
)

func BenchmarkServeConn(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	srv := NewServer("bench")
	srv.Router.GET("/", func(ctx *RequestCtx) {
		ctx.Response.WithText("OK")
	})

	// Start ServeConn in a goroutine
	go srv.ServeConn(serverConn)

	reqStr := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	reader := bufio.NewReader(clientConn)

	for b.Loop() {
		// Write request
		_, err := clientConn.Write([]byte(reqStr))
		if err != nil {
			b.Fatalf("write error: %v", err)
		}
		// Read response
		resp, err := http.ReadResponse(reader, nil)
		if err != nil {
			b.Fatalf("read error: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
