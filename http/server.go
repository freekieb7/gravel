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
	WorkerPoolSize          = 1024 // must be power of 2
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

type Server struct {
	Name         string
	Handler      Handler
	ShutdownFunc func(context.Context) error
	WorkerPool   WorkerPool
	connCh       chan net.Conn
}

func NewServer(name string, handler Handler) Server {
	s := Server{
		Name:         name,
		Handler:      handler,
		ShutdownFunc: func(ctx context.Context) error { return nil },
		WorkerPool:   NewWorkerPool(handler),
		connCh:       make(chan net.Conn, WorkerPoolSize),
	}
	// Start workers
	for range WorkerPoolSize {
		go s.worker()
	}
	return s
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
			// Handle listener closure gracefully
			if _, ok := err.(net.Error); ok {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if err == io.EOF {
				return nil
			}
			// Optionally log the error here
			return err
		}

		// Optionally set a deadline on the connection
		// conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Non-blocking send to worker pool, close conn if full
		select {
		case s.connCh <- conn:
			// Successfully dispatched to worker
		default:
			// All workers are busy, reject connection
			conn.Close()
			// Optionally log: "connection dropped: worker pool full"
		}
	}
}

// Worker goroutine
func (s *Server) worker() {
	for conn := range s.connCh {
		s.ServeConn(conn)
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
		v, found := reqCtx.Request.HeaderValue(headerConnection)
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
