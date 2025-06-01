package http

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/freekieb7/gravel/telemetry"
)

const (
	MaxRequestSize          = 2 * 1024 * 1024 // 2MB
	DefaultBufferReaderSize = 2 * 1024 * 2    // 4kB
	DefaultReadBufferSize   = 4096            // 4kB
	DefaultWriteBufferSize  = 4096            // 4kB
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

func (s *Server) AcquireCtx(c net.Conn) (ctx *RequestCtx) {
	v := s.RequestCtxPool.Get()
	if v == nil {
		ctx = new(RequestCtx)
		ctx.Reset()
	} else {
		ctx = v.(*RequestCtx)
	}

	ctx.Conn = c

	return ctx
}

func (s *Server) ReleaseCtx(ctx *RequestCtx) {
	ctx.Reset()
	s.RequestCtxPool.Put(ctx)
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	// Setup opentelemetry
	otelShutdown, err := telemetry.Setup(ctx)
	if err != nil {
		return err
	}
	s.ShutdownFunc = otelShutdown

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	workerPool := WorkerPool{
		Pool: sync.Pool{
			New: func() any {
				return &Worker{
					ConnCh:   make(chan net.Conn),
					WorkFunc: s.ServeConn,
				}
			},
		},
		StopChan: make(chan struct{}),
	}
	go workerPool.Start()

	for {
		conn, err := listener.Accept()
		if err != nil {
			workerPool.Stop()
			fmt.Printf("failed to accept connection: %s", err)
			continue
		}

		worker := workerPool.Pool.Get().(*Worker)
		go worker.Start()

		worker.ConnCh <- conn
	}
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.ShutdownFunc(ctx)
}

func (server *Server) ServeConn(c net.Conn) (err error) {
	reader := bufio.NewReaderSize(c, DefaultReadBufferSize)
	writer := bufio.NewWriterSize(c, DefaultWriteBufferSize)

	reqCtx := server.AcquireCtx(c)

	for {
		err := reqCtx.Request.Read(reader)
		if err != nil {
			break
		}

		handleFunc := NotFoundHandleFunc
		for _, route := range server.Router.Routes {
			if route.Path == reqCtx.Request.Path {
				handleFunc = route.HandleFunc
				break
			}
		}

		handleFunc(reqCtx)

		if reqCtx.Request.KeepAlive {
			reqCtx.Response.Headers["Connection"] = []string{"keep-alive"}
		} else {
			reqCtx.Response.Headers["Connection"] = []string{"close"}
		}

		if err := reqCtx.Response.Write(writer); err != nil {
			break
		}

		if !reqCtx.Request.KeepAlive {
			break
		}

		c.SetDeadline(time.Now().Add(time.Second * 5))
	}

	server.ReleaseCtx(reqCtx)
	return err
}

type WorkerPool struct {
	Pool       sync.Pool
	WorkerFunc func(c net.Conn)

	StopChan chan struct{}
}

func (wp *WorkerPool) Start() {
	for {
		select {
		case <-wp.StopChan:
			return
		default:
			time.Sleep(10 * time.Second)
		}
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.StopChan)
}

func (wp *WorkerPool) Serve(conn net.Conn) {
	worker := wp.Pool.Get().(*Worker)
	worker.Start()

	worker.ConnCh <- conn
}

type Worker struct {
	ConnCh   chan net.Conn
	WorkFunc func(c net.Conn) error
}

func (w *Worker) Start() {
	for conn := range w.ConnCh {
		if conn == nil {
			break
		}

		w.WorkFunc(conn)
	}
}
