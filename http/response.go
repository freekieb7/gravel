package http

import (
	"bufio"
	"encoding/json"
	"errors"
)

type Response struct {
	Status      uint16
	KeepAlive   bool
	Body        []byte
	Chunked     bool // New field for chunked encoding
	headerBuf   [1024]byte
	headers     [16]Header
	headerCount int
	// Buffer for chunk size hex conversion
	chunkSizeBuf [16]byte
	// Add internal writer reference for streaming
	writer *bufio.Writer
}

func (res *Response) Reset() {
	// Don't zero the entire struct - just reset critical fields
	res.Status = StatusOK
	res.KeepAlive = true
	res.Body = nil
	res.headerCount = 0
	res.Chunked = false
	res.writer = nil // Clear writer reference
}

func (res *Response) SetHeader(name, value []byte) {
	if res.headerCount >= len(res.headers) {
		return // Skip if we've reached max headers
	}

	h := &res.headers[res.headerCount]

	// Copy name (truncate if too long)
	h.NameLen = min(len(name), len(h.Name))
	copy(h.Name[:h.NameLen], name[:h.NameLen])

	// Copy value (truncate if too long)
	h.ValueLen = min(len(value), len(h.Value))
	copy(h.Value[:h.ValueLen], value[:h.ValueLen])

	res.headerCount++
}

func (res *Response) SetHeaderString(name, value string) {
	res.SetHeader([]byte(name), []byte(value))
}

// SetCookie adds a Set-Cookie header to the response
func (res *Response) SetCookie(cookie *Cookie) {
	if cookie == nil {
		return
	}

	// Validate cookie before setting
	if err := cookie.Valid(); err != nil {
		return // Skip invalid cookies
	}

	cookieStr := cookie.String()
	res.SetHeaderString("Set-Cookie", cookieStr)
}

// SetCookieValue is a convenience method to set a simple cookie
func (res *Response) SetCookieValue(name, value string) {
	cookie := &Cookie{
		Name:  name,
		Value: value,
		Path:  "/",
	}
	res.SetCookie(cookie)
}

// DeleteCookie marks a cookie for deletion
func (res *Response) DeleteCookie(name string) {
	cookie := &Cookie{
		Name:   name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	res.SetCookie(cookie)
}

func (res *Response) WithText(payload string) *Response {
	res.Body = []byte(payload)
	res.SetHeaderString("content-type", "text/plain")
	return res
}

func (res *Response) WithJSON(payload any) *Response {
	switch p := payload.(type) {
	case string:
		res.Body = []byte(p)
	case []byte:
		res.Body = p
	default:
		data, _ := json.Marshal(p)
		res.Body = data
	}

	res.SetHeaderString("content-type", "application/json")
	return res
}

// WithHTML sets HTML content with appropriate content type
func (res *Response) WithHTML(payload string) *Response {
	res.Body = []byte(payload)
	res.SetHeaderString("content-type", "text/html; charset=utf-8")
	return res
}

// WithXML sets XML content with appropriate content type
func (res *Response) WithXML(payload string) *Response {
	res.Body = []byte(payload)
	res.SetHeaderString("content-type", "application/xml; charset=utf-8")
	return res
}

// WithFile sets response for file download
func (res *Response) WithFile(filename string, data []byte, contentType string) *Response {
	res.Body = data
	if contentType != "" {
		res.SetHeaderString("content-type", contentType)
	}
	res.SetHeaderString("content-disposition", "attachment; filename=\""+filename+"\"")
	return res
}

// WithRedirect sets up a redirect response
func (res *Response) WithRedirect(location string, statusCode uint16) *Response {
	res.Status = statusCode
	res.SetHeaderString("location", location)
	res.Body = nil
	return res
}

// WriteChunk writes a chunk for chunked transfer encoding
func (res *Response) WriteChunk(data []byte) error {
	if res.writer == nil {
		return errors.New("response writer not available")
	}

	if !res.Chunked {
		return errors.New("response not in chunked mode")
	}

	// Write chunk size in hex
	chunkSize := len(data)
	if chunkSize == 0 {
		// End chunk
		_, err := res.writer.WriteString("0\r\n\r\n")
		return err
	}

	// Convert size to hex and write
	hexLen := writeHexToBuffer(chunkSize, res.chunkSizeBuf[:])
	if _, err := res.writer.Write(res.chunkSizeBuf[:hexLen]); err != nil {
		return err
	}
	if _, err := res.writer.WriteString("\r\n"); err != nil {
		return err
	}
	if _, err := res.writer.Write(data); err != nil {
		return err
	}
	if _, err := res.writer.WriteString("\r\n"); err != nil {
		return err
	}

	return res.writer.Flush()
}

// StartChunkedResponse initializes chunked transfer encoding
func (res *Response) StartChunkedResponse() {
	res.Chunked = true
	res.SetHeaderString("transfer-encoding", "chunked")
}

func (res *Response) WriteTo(bw *bufio.Writer) error {
	// Fast path for empty body responses (no chunking needed)
	if len(res.Body) == 0 && res.headerCount == 0 && res.Status == StatusOK && !res.Chunked {
		if res.KeepAlive {
			if _, err := bw.Write(response200Empty); err != nil {
				return err
			}
		} else {
			if _, err := bw.Write(response200Close); err != nil {
				return err
			}
		}
		return nil
	}

	// Build headers
	var n int

	// Write status line
	if res.Status == StatusOK {
		n += copy(res.headerBuf[n:], http200OK)
	} else {
		n += copy(res.headerBuf[n:], "HTTP/1.1 ")
		n += writeIntToBuffer(int(res.Status), res.headerBuf[n:])
		if message := statusMessages[res.Status]; message != "" {
			n += copy(res.headerBuf[n:], " "+message)
		} else {
			n += copy(res.headerBuf[n:], " Unknown")
		}
		n += copy(res.headerBuf[n:], "\r\n")
	}

	// Write Connection header
	if res.KeepAlive {
		n += copy(res.headerBuf[n:], connectionKeepAlive)
	} else {
		n += copy(res.headerBuf[n:], connectionClose)
	}

	// Write Transfer-Encoding or Content-Length
	if res.Chunked {
		n += copy(res.headerBuf[n:], headerTransferEncodingChunked)
	} else {
		n += copy(res.headerBuf[n:], contentLengthPrefix)
		n += writeIntToBuffer(len(res.Body), res.headerBuf[n:])
		n += copy(res.headerBuf[n:], "\r\n")
	}

	// Write custom headers
	for i := 0; i < res.headerCount; i++ {
		h := &res.headers[i]
		n += copy(res.headerBuf[n:], h.Name[:h.NameLen])
		n += copy(res.headerBuf[n:], ": ")
		n += copy(res.headerBuf[n:], h.Value[:h.ValueLen])
		n += copy(res.headerBuf[n:], "\r\n")
	}

	// End headers
	n += copy(res.headerBuf[n:], "\r\n")

	// Write all headers at once
	if _, err := bw.Write(res.headerBuf[:n]); err != nil {
		return err
	}

	// Write body - chunked or regular
	if res.Chunked {
		if len(res.Body) > 0 {
			if err := res.writeChunk(bw, res.Body); err != nil {
				return err
			}
		}
		if err := res.writeChunkEnd(bw); err != nil {
			return err
		}
	} else {
		if len(res.Body) > 0 {
			if _, err := bw.Write(res.Body); err != nil {
				return err
			}
		}
	}

	return bw.Flush()
}

func (res *Response) writeHeaders(bw *bufio.Writer) error {
	var n int

	// Write status line
	if res.Status == StatusOK {
		n += copy(res.headerBuf[n:], http200OK)
	} else {
		n += copy(res.headerBuf[n:], "HTTP/1.1 ")
		n += writeIntToBuffer(int(res.Status), res.headerBuf[n:])
		n += copy(res.headerBuf[n:], " OK\r\n") // Simplified
	}

	// Write Connection header
	if res.KeepAlive {
		n += copy(res.headerBuf[n:], connectionKeepAlive)
	} else {
		n += copy(res.headerBuf[n:], connectionClose)
	}

	// Write Transfer-Encoding
	n += copy(res.headerBuf[n:], headerTransferEncodingChunked)

	// Write custom headers
	for i := 0; i < res.headerCount; i++ {
		h := &res.headers[i]
		n += copy(res.headerBuf[n:], h.Name[:h.NameLen])
		n += copy(res.headerBuf[n:], ": ")
		n += copy(res.headerBuf[n:], h.Value[:h.ValueLen])
		n += copy(res.headerBuf[n:], "\r\n")
	}

	// End headers
	n += copy(res.headerBuf[n:], "\r\n")

	if _, err := bw.Write(res.headerBuf[:n]); err != nil {
		return err
	}

	return nil
}

func (res *Response) writeChunk(bw *bufio.Writer, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Write chunk size in hex
	hexLen := writeHexToBuffer(len(data), res.chunkSizeBuf[:])
	if _, err := bw.Write(res.chunkSizeBuf[:hexLen]); err != nil {
		return err
	}
	if _, err := bw.Write(crlfOnly); err != nil {
		return err
	}

	// Write chunk data
	if _, err := bw.Write(data); err != nil {
		return err
	}
	if _, err := bw.Write(crlfOnly); err != nil {
		return err
	}

	return nil
}

func (res *Response) writeChunkEnd(bw *bufio.Writer) error {
	if _, err := bw.Write(chunkEndBytes); err != nil {
		return err
	}

	return nil
}

func (res *Response) StartChunked(bw *bufio.Writer) (*ChunkWriter, error) {
	res.Chunked = true
	res.Body = nil // Clear body since we're streaming

	// Write headers first
	if err := res.writeHeaders(bw); err != nil {
		return nil, err
	}

	return &ChunkWriter{bw: bw, res: res}, nil
}

// ChunkWriter allows streaming responses
type ChunkWriter struct {
	bw  *bufio.Writer
	res *Response
}

func (cw *ChunkWriter) WriteChunk(data []byte) error {
	return cw.res.writeChunk(cw.bw, data)
}

func (cw *ChunkWriter) Close() error {
	return cw.res.writeChunkEnd(cw.bw)
}

// Add streaming methods to ChunkWriter
func (cw *ChunkWriter) Write(data []byte) (int, error) {
	if err := cw.WriteChunk(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (cw *ChunkWriter) Flush() error {
	return cw.bw.Flush()
}

// StreamingResponse provides a streaming interface
type StreamingResponse struct {
	writer *ChunkWriter
	bw     *bufio.Writer
}

func (res *Response) StartStreaming() (*StreamingResponse, error) {
	if res.writer == nil {
		return nil, errors.New("response not associated with connection")
	}

	cw, err := res.StartChunked(res.writer)
	if err != nil {
		return nil, err
	}

	return &StreamingResponse{
		writer: cw,
		bw:     res.writer,
	}, nil
}

func (sr *StreamingResponse) WriteString(data string) error {
	return sr.writer.WriteChunk([]byte(data))
}

func (sr *StreamingResponse) Write(data []byte) (int, error) {
	if err := sr.writer.WriteChunk(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (sr *StreamingResponse) WriteJSON(data string) error {
	return sr.writer.WriteChunk([]byte(data))
}

func (sr *StreamingResponse) Flush() error {
	return sr.bw.Flush()
}

func (sr *StreamingResponse) Close() error {
	if err := sr.writer.Close(); err != nil {
		return err
	}
	return sr.bw.Flush()
}

func (r *Response) AddCookie(cookie Cookie) {
	// Validate cookie
	if cookie.Name == "" {
		return
	}

	// For cookies, we always add a new Set-Cookie header (don't overwrite)
	cookieStr := cookie.String()
	r.addHeader([]byte("Set-Cookie"), []byte(cookieStr))
}

// Helper method to add headers without overwriting (for Set-Cookie)
func (r *Response) addHeader(name, value []byte) {
	if r.headerCount >= len(r.headers) {
		return // Skip if we've reached max headers
	}

	h := &r.headers[r.headerCount]

	// Copy name (truncate if too long)
	h.NameLen = min(len(name), len(h.Name))
	copy(h.Name[:h.NameLen], name[:h.NameLen])

	// Copy value (truncate if too long)
	h.ValueLen = min(len(value), len(h.Value))
	copy(h.Value[:h.ValueLen], value[:h.ValueLen])

	r.headerCount++
}
