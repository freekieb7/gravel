package http

import (
	"bufio"
	"context"
	"io"
	"net"
)

type worker struct {
	incommingConnectionChannel chan net.Conn
}

const (
	maxRequestSize          = 2 * 1024 * 1024 // 2MB
	defaultBufferReaderSize = 2 * 1024 * 2    // 4kB
)

func (w *worker) Start(ctx context.Context) {
	go func(ctx context.Context) {
		buffer := make([]byte, defaultBufferReaderSize)

		for {
			select {
			case connection := <-w.incommingConnectionChannel:
				{
					bf := bufio.NewReader(connection)
					for {
						n, err := bf.Read(buffer)
						// n, err := connection.Read(buffer)

						if err == io.EOF {
							connection.Write([]byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\n404 Not Found"))
							connection.Close()
							break
						}

						if err != nil {
							panic(err)
						}

						if n == defaultBufferReaderSize {
							continue
						}

						connection.Write([]byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\n404 Not Found"))
						connection.Close()
						break
					}

				}
			case <-ctx.Done():
				{
					return
				}
			}

		}
	}(ctx)
}

func Serve(ctx context.Context, listener net.Listener) error {
	worker := worker{
		incommingConnectionChannel: make(chan net.Conn),
	}
	worker.Start(ctx)

	for {
		accept, err := listener.Accept()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		worker.incommingConnectionChannel <- accept
	}
}
