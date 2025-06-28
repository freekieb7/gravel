package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"time"
)

const (
	MaxRequestSize          = 2 * 1024 * 1024 // 2MB
	MaxResponseSize         = 2 * 1024 * 1024 // 2MB
	DefaultBufferReaderSize = 2 * 1024 * 2    // 4kB
	DefaultReadBufferSize   = 4096            // 4kB
	DefaultWriteBufferSize  = 4096            // 4kB
	MaxRequestHeaders       = math.MaxUint8
	MaxResponseHeaders      = math.MaxUint8
)

var (
	Http11                = []byte("HTTP/1.1")
	headerConnection      = []byte("Connection")
	headerKeepAlive       = []byte("keep-alive")
	headerClose           = []byte("close")
	headerContentType     = []byte("Content-Type")
	headerApplicationJSON = []byte("application/json")
	// ...add others as needed
)

const DefaultConcurrency uint64 = 256 * 1024

type Server struct {
	Name         string
	Handler      Handler
	ShutdownFunc func(context.Context) error
	WorkerPool   WorkerPool
}

func NewServer(name string, handler Handler, concurrency uint64) Server {
	return Server{
		Name:         name,
		Handler:      handler,
		ShutdownFunc: func(ctx context.Context) error { return nil },
		WorkerPool:   NewWorkerPool(handler, concurrency),
	}
}

func (s *Server) ListenAndServe(addr string) error {
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
			if err == io.EOF {
				return nil
			}
			return err

		}

		go s.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn net.Conn) error {
	defer conn.Close()

	// Search for available worker
	var reqCtx *RequestCtx
	var err error
	for {
		reqCtx, err = s.WorkerPool.Ready.Dequeue()
		if err != nil {
			if err == ErrEmpty {
				time.Sleep(10 * time.Nanosecond)
				continue
				// todo max attempts
			}

			panic(err)
		}

		break
	}
	defer s.WorkerPool.Ready.Enqueue(reqCtx)

	reqCtx.Reset(conn)
	for {
		err := reqCtx.Request.Parse(reqCtx.ConnReader)
		if err != nil {
			if err != io.EOF {
				fmt.Print(err)
			}
			break
		}

		s.Handler(reqCtx)

		// Manage keep alive
		var keepAlive bool
		v, found := reqCtx.Request.HeaderValue("Connection")
		if found {
			keepAlive = bytes.Equal(v, headerKeepAlive)
		} else {
			keepAlive = bytes.Equal(reqCtx.Request.Protocol, Http11)
		}

		if keepAlive {
			reqCtx.Response.SetHeader(headerConnection, headerKeepAlive)
		} else {
			reqCtx.Response.SetHeader(headerConnection, headerClose)
		}

		if err := reqCtx.Response.Write(reqCtx.ConnWriter); err != nil {
			break
		}

		// Kill if keep alive is not re
		if !keepAlive {
			break
		}

		// conn.SetDeadline(time.Now().Add(time.Second * 5))
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.ShutdownFunc(ctx)
}
