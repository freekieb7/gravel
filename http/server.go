package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

const (
	MaxRequestSize          = 2 * 1024 * 1024 // 2MB
	DefaultBufferReaderSize = 2 * 1024 * 2    // 4kB
	DefaultReadBufferSize   = 4096            // 4kB
	DefaultWriteBufferSize  = 4096            // 4kB
	MaxRequestHeaders       = math.MaxUint8
)

type Server struct {
	Name         string
	Router       Router
	ShutdownFunc func(context.Context) error

	RequestCtxPool sync.Pool
}

func NewServer(name string) Server {
	return Server{
		Name:         name,
		Router:       NewRouter(),
		ShutdownFunc: func(ctx context.Context) error { return nil },

		RequestCtxPool: sync.Pool{},
	}
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(listener)
}

func (s *Server) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection %v", err)
			continue

		}

		go s.ServeConn(conn) // scale up to n workers
	}
}

func (s *Server) ServeConn(conn net.Conn) {
	defer conn.Close()

	br := bufio.NewReaderSize(conn, DefaultReadBufferSize)
	bw := bufio.NewWriterSize(conn, DefaultWriteBufferSize)

	reqCtx := RequestCtx{
		Request: Request{},
		Response: Response{
			Status:  200,
			Headers: Headers{},
			Body:    make([]byte, 0),
		},
	}

	for {
		err := reqCtx.Request.Parse(br)
		if err != nil {
			if err != io.EOF {
				fmt.Print(err)
			}
			break
		}

		handleFunc := NotFoundHandleFunc
		for _, route := range s.Router.Routes {
			if route.Path != string(reqCtx.Request.Path) {
				continue
			}

			for _, method := range route.Methods {
				if method != string(reqCtx.Request.Method) {
					continue
				}

				handleFunc = route.HandleFunc
				break
			}
		}

		handleFunc(&reqCtx)

		var keepAlive bool
		v, found := reqCtx.Request.HeaderValue("Connection")
		if found {
			keepAlive = bytes.Equal(v, []byte("keep-alive"))
		} else {
			keepAlive = bytes.Equal(reqCtx.Request.Protocol, []byte("HTTP/1.1"))
		}

		if keepAlive {
			reqCtx.Response.Headers["Connection"] = []string{"keep-alive"}
		} else {
			reqCtx.Response.Headers["Connection"] = []string{"close"}
		}

		if err := reqCtx.Response.Write(bw); err != nil {
			break
		}

		if !keepAlive {
			break
		}

		conn.SetDeadline(time.Now().Add(time.Second * 5))
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.ShutdownFunc(ctx)
}
