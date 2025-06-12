package http

import (
	"bufio"
	"bytes"
	"testing"
)

func TestRequestParse(t *testing.T) {
	var req Request

	reqMsg := []byte("GET /test HTTP/1.1\r\nAccept: text/css\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n")

	br := bufio.NewReader(bytes.NewBuffer(reqMsg))

	if err := req.Parse(br); err != nil {
		t.Error(err)
	}

	h, found := req.HeaderValue("connection")
	if !found {
		t.Error("connection header not found")
	}
	if !bytes.Equal(h, []byte("keep-alive")) {
		t.Errorf("expected keep-alive, got %s", h)
	}
}

func BenchmarkRequestParse(b *testing.B) {
	reqMsg := []byte("GET /test HTTP/1.1\r\nAccept: text/css\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n")
	var req Request

	reader := bytes.NewReader(reqMsg)
	br := bufio.NewReader(reader)

	for b.Loop() {
		reader.Reset(reqMsg) // Reset read position without allocation
		br.Reset(reader)     // Reset bufio.Reader to reuse buffer

		if err := req.Parse(br); err != nil {
			b.Error(err)
		}
	}
}
