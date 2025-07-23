package http

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"net"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	WorkerPoolSize uint32
	Handler        func(req *Request, res *Response)
	ShutdownCh     chan struct{}
	Wg             sync.WaitGroup // Registers shutdowns
}

func NewServer(handler Handler) Server {
	return Server{
		Handler:    handler,
		ShutdownCh: make(chan struct{}),
	}
}

func (s *Server) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// Optimize TCP listener
	if tcpLn, ok := ln.(*net.TCPListener); ok {
		// Set socket options for better performance
		if err := tcpLn.SetDeadline(time.Time{}); err != nil {
			return err
		}
	}

	return s.Serve(ln)
}

func (s *Server) Serve(ln net.Listener) error {
	defer func() {
		if err := ln.Close(); err != nil {
			slog.Error("ln.Close error", "error", err)
		}
	}()

	// Add this defer to signal completion of Serve method
	defer s.Wg.Done()

	// Auto-size worker pool if not set - make it power of 2 for faster modulo
	if s.WorkerPoolSize == 0 {
		cores := uint32(runtime.NumCPU())
		s.WorkerPoolSize = 1
		for s.WorkerPoolSize < cores*512 {
			s.WorkerPoolSize <<= 1 // Next power of 2
		}
	}

	workerChannels := make([]chan net.Conn, s.WorkerPoolSize)
	for i := range workerChannels {
		s.Wg.Add(1) // This is for each worker goroutine

		workerChannels[i] = make(chan net.Conn, ChannelBufferSize)
		go s.ServeConn(workerChannels[i])
	}

	// Use atomic operations for better performance
	var counter uint32
	mask := s.WorkerPoolSize - 1 // For power-of-2 fast modulo

	for {
		select {
		case <-s.ShutdownCh:
			log.Println("Server shutdown initiated...")

			// Close all worker channels to signal shutdown
			for i := range workerChannels {
				close(workerChannels[i])
			}

			return nil
		default:
		}

		// Set a short timeout for Accept during shutdown
		if tcpLn, ok := ln.(*net.TCPListener); ok {
			if err := tcpLn.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				slog.Error("SetDeadline error", "error", err)
			}
		}

		conn, err := ln.Accept()
		if err != nil {
			// Check if this is due to shutdown
			select {
			case <-s.ShutdownCh:
				log.Println("Server shutdown during Accept")
				for i := range workerChannels {
					close(workerChannels[i])
				}
				return nil
			default:
				// Check if it's a timeout (expected during shutdown)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Continue the loop to check shutdown again
				}
				return err
			}
		}

		// Reset deadline after successful accept
		if tcpLn, ok := ln.(*net.TCPListener); ok {
			if err := tcpLn.SetDeadline(time.Time{}); err != nil {
				return err
			}
		}

		// Fast modulo using bitwise AND (only works with power of 2)
		idx := counter & mask
		counter++

		// Try multiple workers before giving up
		for range 3 {
			select {
			case workerChannels[idx] <- conn:
				goto next_connection
			default:
				idx = (idx + 1) & mask // Try next worker
			}
		}

		// All workers busy
		if err := conn.Close(); err != nil {
			slog.Error("closing connection error", "error", err)
		}

	next_connection:
	}
}

func (s *Server) ServeConn(ch chan net.Conn) {
	// Signal completion when this worker exits
	defer s.Wg.Done()

	// Shutdown / crash behaviour
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered", "error", r)
		}
	}()

	br := bufio.NewReaderSize(nil, DefaultReadBufferSize)
	bw := bufio.NewWriterSize(nil, DefaultWriteBufferSize)

	req := Request{}
	res := Response{}

	for conn := range ch {
		// Check if channel was closed (shutdown signal)
		if conn == nil {
			return
		}

		req.Method = nil
		req.Path = nil
		req.Protocol = nil
		req.Body = nil
		req.Close = false

		// Don't zero the entire struct - just reset critical fields
		res.Status = StatusOK
		res.KeepAlive = true
		res.Body = nil
		res.headerCount = 0
		res.Chunked = false
		res.writer = nil // Clear writer reference

		s.handleConnection(conn, br, bw, &req, &res)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Send shutdown signal to all workers
	close(s.ShutdownCh)

	done := make(chan struct{})
	go func() {
		// Wait for all workers to acknowledge shutdown
		s.Wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) handleConnection(conn net.Conn, br *bufio.Reader, bw *bufio.Writer, req *Request, res *Response) {
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("closing connection error", "error", err)
		}
	}()

	// Optimize TCP connection once per connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetNoDelay(true); err != nil { // Disable Nagle's algorithm
			// Log but don't fail - this is an optimization
			slog.Error("SetNoDelay error", "error", err)
		}
		if err := tcpConn.SetKeepAlive(true); err != nil { // Enable keep-alive
			// Log but don't fail - this is an optimization
			slog.Error("SetKeepAlive error", "error", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(3 * time.Minute); err != nil { // Set longer keep-alive period
			// Log but don't fail - this is an optimization
			slog.Error("SetKeepAlivePeriod error", "error", err)
		}
		if err := tcpConn.SetReadBuffer(128 * 1024); err != nil { // Set larger read buffer for better performance
			// Log but don't fail - this is an optimization
			slog.Error("SetReadBuffer error", "error", err)
		}
		if err := tcpConn.SetWriteBuffer(128 * 1024); err != nil { // Set larger write buffer for better performance
			// Log but don't fail - this is an optimization
			slog.Error("SetWriteBuffer error", "error", err)
		}
	}

	br.Reset(conn)
	bw.Reset(conn)

	// Reduced connection timeout for faster shutdown
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		slog.Error("SetDeadline error", "error", err)
	}

	requestCount := 0
	// Reduced max requests per connection for faster shutdown
	maxRequestsPerConnection := 10000

	for requestCount < maxRequestsPerConnection {
		requestCount++

		// Reset response fields individually instead of struct copy
		res.Status = StatusOK
		res.KeepAlive = true
		res.Body = nil
		res.headerCount = 0
		res.Chunked = false
		// Associate writer with response
		res.writer = bw

		// Check for shutdown every request
		select {
		case <-s.ShutdownCh:
			// Send connection close response and exit
			res.KeepAlive = false
			res.Body = []byte("Server shutting down")
			if err := res.WriteTo(bw); err != nil {
				slog.Error("WriteTo error", "error", err)
			}
			return
		default:
		}

		if err := req.Parse(br); err != nil {
			if err == io.EOF {
				break
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}

			if errors.Is(err, syscall.ECONNRESET) {
				break
			}

			log.Print("Parse error:", err)
			break
		}

		res.KeepAlive = !req.Close

		// Call handler - it can now use streaming without bw parameter
		s.Handler(req, res)

		// Only write response if not already handled by streaming
		if !res.Chunked || res.Body != nil {
			if err := res.WriteTo(bw); err != nil {
				log.Print("WriteTo error:", err)
				break
			}
		}

		// Clear writer reference for safety
		res.writer = nil

		if req.Close {
			break
		}

		// Update deadline more frequently for faster shutdown
		if requestCount%5 == 0 {
			if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				slog.Error("SetDeadline error", "error", err)
			}
		}
	}

	// Final flush for any remaining data
	if err := bw.Flush(); err != nil {
		slog.Error("Flush error", "error", err)
	}
}
