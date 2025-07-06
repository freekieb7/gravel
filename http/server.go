package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"time"
)

const (
	DefaultReadBufferSize  = 2 * 1024 * 1024
	DefaultWriteBufferSize = 2 * 1024 * 1024
	MaxRequestHeaders      = 32
	MaxResponseHeaders     = 32
	// WorkerPoolSize         uint32 = 2 * runtime.NumCPU() // or higher, depending on your CPU and workload
)

const WorkerPoolSize uint32 = 16

var (
	Http11           = []byte("HTTP/1.1")
	headerConnection = []byte("connection")
	headerKeepAlive  = []byte("keep-alive")
	headerClose      = []byte("close")
)

type Server struct {
	Name         string
	Handler      Handler
	ShutdownFunc func(context.Context) error
	shutdownCh   chan struct{}
}

func NewServer(name string, handler Handler) *Server {
	return &Server{
		Name:         name,
		Handler:      handler,
		ShutdownFunc: func(ctx context.Context) error { return nil },
		shutdownCh:   make(chan struct{}),
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

	var workers [WorkerPoolSize]chan net.Conn
	for i := range workers {
		workers[i] = make(chan net.Conn, 10000)

		go func(connChan chan net.Conn, handler Handler) {
			var (
				br = bufio.NewReaderSize(nil, DefaultReadBufferSize)
				bw = bufio.NewWriterSize(nil, DefaultWriteBufferSize)
			)

			var requestCtx RequestCtx

			for conn := range connChan {
				br.Reset(conn)
				bw.Reset(conn)

				for {
					requestCtx.Request.Reset()
					requestCtx.Response.Reset()

					conn.SetReadDeadline(time.Now().Add(5 * time.Second))

					// Handle request
					err := requestCtx.Request.Parse(br)
					if err != nil {
						break
					}

					handler(&requestCtx)

					// Keep-alive
					var keepAlive bool
					v, found := requestCtx.Request.HeaderValue(headerConnection)
					if found {
						keepAlive = bytes.Equal(v, headerKeepAlive)
					} else {
						keepAlive = bytes.Equal(requestCtx.Request.Protocol, Http11)
					}

					if keepAlive {
						requestCtx.Response.SetHeader(headerConnection, headerKeepAlive)
					} else {
						requestCtx.Response.SetHeader(headerConnection, headerClose)
					}

					// Protection against slow client acknowledgements
					conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
					if err := requestCtx.Response.Write(bw); err != nil {
						break
					}
					if err := bw.Flush(); err != nil {
						break
					}
					conn.SetWriteDeadline(time.Time{})

					if !keepAlive {
						break
					}
				}

				conn.Close()
			}
		}(workers[i], s.Handler)
	}

	var counter uint32
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || err == io.EOF {
				return nil
			}
			return err
		}

		idx := counter % WorkerPoolSize
		counter++

		select {
		case workers[idx] <- conn:
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
		// s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
