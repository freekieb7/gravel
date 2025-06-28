package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxRequestSize         = 2 * 1024 // 2MB
	MaxResponseSize        = 2 * 1024 // 2MB
	DefaultReadBufferSize  = 4096
	DefaultWriteBufferSize = 4096
	MaxRequestHeaders      = 255
	MaxResponseHeaders     = 255
	WorkerPoolSize         = 4096 // or higher, depending on your CPU and workload
)

var (
	Http11                = []byte("HTTP/1.1")
	headerConnection      = []byte("connection")
	headerKeepAlive       = []byte("keep-alive")
	headerClose           = []byte("close")
	headerContentType     = []byte("content-type")
	headerApplicationJSON = []byte("application/json")
)

type worker struct {
	connCh  chan net.Conn
	reqCtx  *RequestCtx
	handler Handler
}

type Server struct {
	Name         string
	Handler      Handler
	ShutdownFunc func(context.Context) error
	workers      []*worker
	next         uint32 // for round-robin
	wg           sync.WaitGroup
}

func NewServer(name string, handler Handler) *Server {
	s := &Server{
		Name:         name,
		Handler:      handler,
		ShutdownFunc: func(ctx context.Context) error { return nil },
		workers:      make([]*worker, WorkerPoolSize),
	}
	for i := range WorkerPoolSize {
		s.wg.Add(1)
		w := &worker{
			connCh: make(chan net.Conn, 1024), // increase from 128 to 1024 or more
			reqCtx: &RequestCtx{
				ConnReader: bufio.NewReaderSize(nil, MaxRequestSize),
				ConnWriter: bufio.NewWriterSize(nil, MaxResponseSize),
			}, // preallocate
			handler: handler,
		}
		go func(w *worker) {
			defer s.wg.Done()
			w.loop()
		}(w)
		s.workers[i] = w
	}
	return s
}

func (w *worker) loop() {
	for conn := range w.connCh {
		w.serveConn(conn)
	}
}

func (w *worker) serveConn(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			// log panic
		}
	}()
	defer conn.Close()
	w.reqCtx.Reset(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		err := w.reqCtx.Request.Parse(w.reqCtx.ConnReader)
		if err != nil {
			break
		}
		w.handler(w.reqCtx)

		// Manage keep-alive
		var keepAlive bool
		v, found := w.reqCtx.Request.HeaderValue(headerConnection)
		if found {
			keepAlive = bytes.Equal(v, headerKeepAlive)
		} else {
			keepAlive = bytes.Equal(w.reqCtx.Request.Protocol, Http11)
		}

		if keepAlive {
			w.reqCtx.Response.SetHeader(headerConnection, headerKeepAlive)
		} else {
			w.reqCtx.Response.SetHeader(headerConnection, headerClose)
		}

		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := w.reqCtx.Response.Write(w.reqCtx.ConnWriter); err != nil {
			break
		}
		conn.SetWriteDeadline(time.Time{})

		if !keepAlive {
			break
		}
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
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || err == io.EOF {
				return nil
			}
			return err
		}
		// Round-robin dispatch (no lock needed for atomic increment)
		idx := atomic.AddUint32(&s.next, 1) % WorkerPoolSize
		select {
		case s.workers[idx].connCh <- conn:
		default:
			conn.Close() // worker busy, drop connection
		}
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	for _, w := range s.workers {
		close(w.connCh)
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
