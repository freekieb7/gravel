package http

// func BenchmarkServerServe(b *testing.B) {
// 	serverConn, clientConn := net.Pipe()
// 	defer serverConn.Close()
// 	defer clientConn.Close()

// 	srv := NewServer("bench")
// 	srv.Router.GET("/", func(ctx *RequestCtx) {
// 		ctx.Response.WithText("OK")
// 	})

// 	// Start server in goroutine
// 	go func() {
// 		worker := NewWorker()

// 		go worker.Start(srv.Handlers())
// 		for {
// 			worker.ConnCh <- serverConn
// 		}
// 	}()

// 	reqStr := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
// 	reader := bufio.NewReader(clientConn)

// 	b.ResetTimer()
// 	for b.Loop() {
// 		// Write request
// 		_, err := clientConn.Write([]byte(reqStr))
// 		if err != nil {
// 			b.Fatalf("write error: %v", err)
// 		}
// 		// Read response
// 		resp, err := http.ReadResponse(reader, nil)
// 		if err != nil {
// 			b.Fatalf("read error: %v", err)
// 		}
// 		io.Copy(io.Discard, resp.Body)
// 	}
// }
