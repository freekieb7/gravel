package http

const (
	DefaultReadBufferSize  = 64 * 1024 // Increase buffer sizes
	DefaultWriteBufferSize = 64 * 1024
	ChannelBufferSize      = 2000 // Increase channel buffer
)

type Handler func(req *Request, res *Response)

var (
	protocolHttp10         = []byte("HTTP/1.0")
	protocolHttp11         = []byte("HTTP/1.1")
	headerContentLength    = []byte("content-length")
	headerTransferEncoding = []byte("transfer-encoding")
	headerConnection       = []byte("connection")
	headerKeepAlive        = []byte("keep-alive")
	headerClose            = []byte("close")
	// Pre-compute common response patterns
	http200OK           = []byte("HTTP/1.1 200 OK\r\n")
	connectionKeepAlive = []byte("connection: keep-alive\r\n")
	connectionClose     = []byte("connection: close\r\n")
	contentLengthPrefix = []byte("content-length: ")
	// Pre-computed complete responses for common cases
	response200Empty = []byte("HTTP/1.1 200 OK\r\nconnection: keep-alive\r\ncontent-length: 0\r\n\r\n")
	response200Close = []byte("HTTP/1.1 200 OK\r\nconnection: close\r\ncontent-length: 0\r\n\r\n")
	// Pre-computed header parts
	headerTransferEncodingChunked = []byte("transfer-encoding: chunked\r\n")
	chunkEndBytes                 = []byte("0\r\n\r\n") // Final chunk
	crlfOnly                      = []byte("\r\n")
)

type Header struct {
	Name     [64]byte  // Fixed size for header name
	Value    [256]byte // Fixed size for header value
	NameLen  int
	ValueLen int
}
