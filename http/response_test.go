package http

import (
	"bufio"
	"bytes"
	"testing"
)

func TestResponseWrite_Basic(t *testing.T) {
	var res Response
	res.Reset()

	res.Status = 200
	res.SetHeader([]byte("content-type"), []byte("text/plain"))
	res.Body = []byte("hello, world!")

	buf := &bytes.Buffer{}
	bw := bufio.NewWriter(buf)

	if err := res.WriteTo(bw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	wantStatus := "HTTP/1.1 200 OK\r\n"
	wantHeader := "content-type: text/plain\r\n"
	wantContentLength := "content-length: 13\r\n"
	wantBody := "hello, world!"

	if !bytes.HasPrefix([]byte(got), []byte(wantStatus)) {
		t.Errorf("missing or incorrect status line: got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte(wantHeader)) {
		t.Errorf("missing or incorrect header: got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte(wantContentLength)) {
		t.Errorf("missing or incorrect content-length: got %q", got)
	}
	if !bytes.HasSuffix([]byte(got), []byte(wantBody)) {
		t.Errorf("missing or incorrect body: got %q", got)
	}
}

func TestResponseWrite_MultipleHeaders(t *testing.T) {
	var res Response
	res.Status = 404
	res.SetHeader([]byte("x-test"), []byte("foo"))
	res.SetHeader([]byte("x-other"), []byte("bar"))
	res.Body = []byte("not found")

	buf := &bytes.Buffer{}
	bw := bufio.NewWriter(buf)

	if err := res.WriteTo(bw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("x-test: foo\r\n")) {
		t.Errorf("missing x-test header: got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("x-other: bar\r\n")) {
		t.Errorf("missing x-other header: got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("HTTP/1.1 404 Not Found\r\n")) {
		t.Errorf("missing status line: got %q", got)
	}
	if !bytes.HasSuffix([]byte(got), []byte("not found")) {
		t.Errorf("missing or incorrect body: got %q", got)
	}
}

func TestResponseWrite_EmptyBody(t *testing.T) {
	var res Response
	res.Status = 204
	res.SetHeader([]byte("x-empty"), []byte("true"))
	res.Body = nil

	buf := &bytes.Buffer{}
	bw := bufio.NewWriter(buf)

	if err := res.WriteTo(bw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("content-length: 0\r\n")) {
		t.Errorf("expected content-length: 0, got %q", got)
	}
	if !bytes.HasSuffix([]byte(got), []byte("\r\n\r\n")) {
		t.Errorf("expected empty body, got %q", got)
	}
}
func BenchmarkResponseWrite(b *testing.B) {
	var res Response
	res.Status = 200
	res.SetHeader([]byte("content-type"), []byte("text/plain"))
	res.SetHeader([]byte("x-bench"), []byte("1"))
	res.Body = []byte("benchmarking response write")

	buf := &bytes.Buffer{}
	bw := bufio.NewWriter(buf)

	for b.Loop() {
		buf.Reset()
		bw.Reset(buf)
		if err := res.WriteTo(bw); err != nil {
			b.Fatal(err)
		}
	}
}
