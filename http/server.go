package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxRequestSize         = 2 * 1024 * 1024 // 2MB
	MaxResponseSize        = 2 * 1024 * 1024 // 2MB
	DefaultReadBufferSize  = 4096
	DefaultWriteBufferSize = 4096
	MaxRequestHeaders      = 255
	MaxResponseHeaders     = 255
	// WorkerPoolSize         uint32 = 2 * runtime.NumCPU() // or higher, depending on your CPU and workload
)

var (
	workerPoolSize   = uint32(2 * runtime.NumCPU()) // or higher, depending on your CPU and workload
	Http11           = []byte("HTTP/1.1")
	headerConnection = []byte("connection")
	headerKeepAlive  = []byte("keep-alive")
	headerClose      = []byte("close")
	// headerContentType     = []byte("content-type")
	// headerApplicationJSON = []byte("application/json")
)

type Server struct {
	Name         string
	Handler      Handler
	ShutdownFunc func(context.Context) error
	// workers      []*Worker
	counter    uint32 // for round-robin
	wg         sync.WaitGroup
	shutdownCh chan struct{}
}

func NewServer(name string, handler Handler) *Server {
	s := &Server{
		Name:         name,
		Handler:      handler,
		ShutdownFunc: func(ctx context.Context) error { return nil },
		// workers:      make([]*Worker, WorkerPoolSize),
		shutdownCh: make(chan struct{}),
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
	defer listener.Close()

	workers := make([]Worker, workerPoolSize)
	for i := range len(workers) {
		s.wg.Add(1)
		worker := Worker{
			connCh: make(chan net.Conn, 10000),
			reqCtx: &RequestCtx{
				ConnReader: *bufio.NewReaderSize(nil, MaxRequestSize),
				ConnWriter: *bufio.NewWriterSize(nil, MaxResponseSize),
			}, // preallocate
			handler: s.Handler,
		}

		go func(worker *Worker) {
			defer s.wg.Done()
			worker.Start()
		}(&worker)
		workers[i] = worker
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || err == io.EOF {
				return nil
			}
			return err
		}

		// Round-robin dispatch (no lock needed for atomic increment)
		idx := atomic.AddUint32(&s.counter, 1)
		worker := workers[idx&(workerPoolSize-1)]

		select {
		case worker.connCh <- conn:
		default:
			conn.Close() // worker busy, drop connection
		}
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Send shutdown signal to all workers
	close(s.shutdownCh)

	done := make(chan struct{})
	go func() {
		// Wait for all workers to acknowledge shutdown
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

type Worker struct {
	connCh     chan net.Conn
	reqCtx     *RequestCtx
	handler    Handler
	shutdownCh chan struct{}
}

func (w *Worker) Start() {
	for {
		select {
		case conn := <-w.connCh:
			if conn == nil {
				log.Println("channel closed")
				return
			}
			w.ServeConn(conn)
		case <-w.shutdownCh:
			log.Println("channel shutdown")
			return
		}
	}
}

func (w *Worker) ServeConn(conn net.Conn) {
	// Prevent crashing worker
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	defer conn.Close()
	w.reqCtx.Reset(conn)

	for {
		w.reqCtx.Request.Reset()
		w.reqCtx.Response.Reset()

		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// Handle request
		err := w.reqCtx.Request.Parse(&w.reqCtx.ConnReader)
		if err != nil {
			break
		}
		w.handler(w.reqCtx)

		// Keep-alive
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

		// Protection against slow client acknowledgements
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := w.reqCtx.Response.Write(&w.reqCtx.ConnWriter); err != nil {
			break
		}
		conn.SetWriteDeadline(time.Time{})

		if !keepAlive {
			break
		}
	}
}
