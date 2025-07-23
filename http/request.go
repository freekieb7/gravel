package http

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type Request struct {
	Method   []byte
	Path     []byte
	Protocol []byte
	Body     []byte
	Close    bool

	// Add buffer to reuse for body reading
	bodyBuf [4096]byte

	headers     [32]Header // Support up to 32 headers
	headerCount int

	queryParams      [32]QueryParam
	queryParamsCount int
}

func (req *Request) Reset() {
	req.Method = nil
	req.Path = nil
	req.Protocol = nil
	req.Body = nil
	req.Close = false

	req.headerCount = 0
	req.queryParamsCount = 0
}

func (req *Request) QueryParam(name []byte) ([]byte, bool) {
	// Convert name to lowercase for case-insensitive comparison
	lowerName := make([]byte, len(name))
	for i, b := range name {
		if b >= 'A' && b <= 'Z' {
			lowerName[i] = b + 32 // Convert to lowercase
		} else {
			lowerName[i] = b
		}
	}

	// Search through stored params
	for i := 0; i < req.headerCount; i++ {
		h := &req.queryParams[i]

		// Check if name length matches
		if h.NameLen != len(lowerName) {
			continue
		}

		// Compare param names (case-insensitive)
		match := true
		for j := 0; j < h.NameLen; j++ {
			if h.Name[j] != lowerName[j] {
				match = false
				break
			}
		}

		if match {
			return h.Value[:h.ValueLen], true
		}
	}

	return nil, false // Param not found
}

func (req *Request) Parse(br *bufio.Reader) error {
	// Use ReadSlice for the request line
	line, err := br.ReadSlice('\n')
	if err != nil {
		return err
	}

	// Fast path for line ending removal
	lineLen := len(line)
	if lineLen >= 2 && line[lineLen-2] == '\r' {
		line = line[:lineLen-2]
	} else if lineLen >= 1 && line[lineLen-1] == '\n' {
		line = line[:lineLen-1]
	}

	// Parse request line with bounds checking
	if len(line) < 14 {
		return errors.New("request line too short")
	}

	// Single pass parsing
	space1 := -1
	space2 := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			if space1 == -1 {
				space1 = i
			} else {
				space2 = i
				break
			}
		}
	}

	if space1 == -1 || space2 == -1 {
		return errors.New("invalid request line format")
	}

	req.Method = line[:space1]
	req.Path = line[space1+1 : space2]
	req.Protocol = line[space2+1:]

	// Path parse
	if questionIdx := bytes.IndexByte(req.Path, '?'); questionIdx >= 0 {
		// Split path and query string
		actualPath := req.Path[:questionIdx]
		queryString := req.Path[questionIdx+1:]
		req.Path = actualPath // Update path to exclude query string

		// Parse query parameters
		if err := req.parseQueryParams(queryString); err != nil {
			return err
		}
	}

	// Fast protocol check
	if len(req.Protocol) == 8 {
		if bytes.Equal(req.Protocol, protocolHttp11) {
			req.Close = false
		} else if bytes.Equal(req.Protocol, protocolHttp10) {
			req.Close = true
		} else {
			return errors.New("unsupported http protocol")
		}
	} else {
		return errors.New("invalid http protocol")
	}

	// ... rest of header parsing stays the same but optimized
	return req.parseHeaders(br)
}

func (req *Request) parseQueryParams(queryString []byte) error {
	if len(queryString) == 0 {
		return nil
	}

	// Split by & to get individual parameters
	start := 0
	for i := 0; i <= len(queryString); i++ {
		if i == len(queryString) || queryString[i] == '&' {
			if i > start && req.queryParamsCount < len(req.queryParams) {
				param := queryString[start:i]
				if err := req.parseQueryParam(param); err != nil {
					return err
				}
			}
			start = i + 1
		}
	}
	return nil
}

func (req *Request) parseQueryParam(param []byte) error {
	if req.queryParamsCount >= len(req.queryParams) {
		return nil // Ignore if we've reached max params
	}

	qp := &req.queryParams[req.queryParamsCount]

	// Find the = separator
	eqIdx := bytes.IndexByte(param, '=')
	if eqIdx < 0 {
		// No value, treat entire param as name with empty value
		qp.NameLen = min(len(param), len(qp.Name))
		copy(qp.Name[:qp.NameLen], param[:qp.NameLen])
		qp.ValueLen = 0
	} else {
		// Split name and value
		name := param[:eqIdx]
		value := param[eqIdx+1:]

		// URL decode and store name
		nameLen, err := req.urlDecode(name, qp.Name[:])
		if err != nil {
			return err
		}
		qp.NameLen = nameLen

		// URL decode and store value
		valueLen, err := req.urlDecode(value, qp.Value[:])
		if err != nil {
			return err
		}
		qp.ValueLen = valueLen
	}

	// Convert name to lowercase for case-insensitive lookup
	for i := 0; i < qp.NameLen; i++ {
		if qp.Name[i] >= 'A' && qp.Name[i] <= 'Z' {
			qp.Name[i] += 32
		}
	}

	req.queryParamsCount++
	return nil
}

func (req *Request) parseHeaders(br *bufio.Reader) error {
	var (
		contentLength       int
		hasContentLength    bool
		hasTransferEncoding bool
		isChunked           bool
	)

	var lowerNameBuf [64]byte // Reusable buffer for header name conversion

	for {
		b, err := br.ReadSlice('\n')
		if err != nil {
			return err
		}

		// Fast line ending removal
		bLen := len(b)
		if bLen >= 2 && b[bLen-2] == '\r' {
			b = b[:bLen-2]
		} else if bLen >= 1 && b[bLen-1] == '\n' {
			b = b[:bLen-1]
		}

		if len(b) == 0 {
			break // End of headers
		}

		// Find colon
		colonIdx := bytes.IndexByte(b, ':')
		if colonIdx < 0 {
			return errors.New("invalid header format")
		}

		name := b[:colonIdx]
		value := b[colonIdx+1:]

		// Skip leading space in value
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		// Store header if we have space
		if req.headerCount < len(req.headers) {
			h := &req.headers[req.headerCount]

			// Store name in lowercase - NO ALLOCATION
			h.NameLen = min(len(name), len(h.Name))
			for i := 0; i < h.NameLen; i++ {
				if name[i] >= 'A' && name[i] <= 'Z' {
					h.Name[i] = name[i] + 32 // Convert to lowercase
				} else {
					h.Name[i] = name[i]
				}
			}

			// Store value as-is
			h.ValueLen = min(len(value), len(h.Value))
			copy(h.Value[:h.ValueLen], value[:h.ValueLen])

			req.headerCount++
		}

		// Process special headers for protocol logic - NO ALLOCATION
		if len(name) <= 20 && len(name) <= len(lowerNameBuf) {
			// Use pre-allocated buffer instead of make() - FIXES ALLOCATION #1 & #2
			lowerName := lowerNameBuf[:len(name)]
			for i, b := range name {
				if b >= 'A' && b <= 'Z' {
					lowerName[i] = b + 32
				} else {
					lowerName[i] = b
				}
			}

			// SIMD-optimized header matching
			switch len(lowerName) {
			case 10: // connection
				if bytes.Equal(lowerName, headerConnection) {
					if equalsFast(value, headerKeepAlive) {
						req.Close = false
					} else if equalsFast(value, headerClose) {
						req.Close = true
					}
				}
			case 14: // content-length
				if bytes.Equal(lowerName, headerContentLength) {
					if hasTransferEncoding && isChunked {
						return errors.New("potential request smuggling")
					}
					hasContentLength = true
					contentLength, err = atoi(value)
					if err != nil {
						return errors.New("invalid content-length")
					}
				}
			case 17: // transfer-encoding
				if bytes.Equal(lowerName, headerTransferEncoding) {
					hasTransferEncoding = true
					if bytes.Contains(value, []byte("chunked")) {
						isChunked = true
						if hasContentLength {
							return errors.New("potential request smuggling")
						}
					}
				}
			}
		}
	}

	// Read body
	if isChunked {
		return req.readChunkedBody(br)
	} else if contentLength > 0 {
		if contentLength <= len(req.bodyBuf) {
			req.Body = req.bodyBuf[:contentLength]
		} else {
			req.Body = make([]byte, contentLength)
		}
		_, err := io.ReadFull(br, req.Body)
		return err
	}

	req.Body = nil
	return nil
}

func (req *Request) readChunkedBody(br *bufio.Reader) error {
	// Use the existing bodyBuf for small bodies, pre-allocate larger buffer
	bodyBuf := req.bodyBuf[:]
	bodyLen := 0
	maxBodySize := cap(bodyBuf)

	// Pre-allocate larger buffer if needed for chunked bodies
	if maxBodySize < 64*1024 {
		// Only allocate once for large chunked bodies
		bodyBuf = make([]byte, 64*1024)
		maxBodySize = len(bodyBuf)
	}

	for {
		// Read chunk size line
		line, err := br.ReadSlice('\n')
		if err != nil {
			return err
		}

		// Remove \r\n
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			line = line[:len(line)-2]
		} else if len(line) >= 1 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		// Parse hex chunk size
		chunkSize := 0
		for _, b := range line {
			if b >= '0' && b <= '9' {
				chunkSize = chunkSize*16 + int(b-'0')
			} else if b >= 'a' && b <= 'f' {
				chunkSize = chunkSize*16 + int(b-'a'+10)
			} else if b >= 'A' && b <= 'F' {
				chunkSize = chunkSize*16 + int(b-'A'+10)
			} else {
				break // Stop at first non-hex character (chunk extensions)
			}
		}

		if chunkSize == 0 {
			// Read final \r\n
			if _, err := br.ReadSlice('\n'); err != nil {
				return err
			}
			break
		}

		// Check if we have enough space in pre-allocated buffer
		if bodyLen+chunkSize > maxBodySize {
			// Body too large for our buffer - fallback to allocation
			largeBody := make([]byte, bodyLen+chunkSize*2) // Pre-allocate extra space
			copy(largeBody[:bodyLen], bodyBuf[:bodyLen])
			bodyBuf = largeBody
			maxBodySize = len(bodyBuf)
		}

		// Read chunk data directly into pre-allocated buffer - NO ALLOCATION
		_, err = io.ReadFull(br, bodyBuf[bodyLen:bodyLen+chunkSize])
		if err != nil {
			return err
		}

		bodyLen += chunkSize

		// Read trailing \r\n
		if _, err := br.ReadSlice('\n'); err != nil {
			return err
		}
	}

	// Set body to the used portion of the buffer
	req.Body = bodyBuf[:bodyLen]
	return nil
}

// Add Header method to retrieve header value
func (req *Request) Header(name []byte) ([]byte, bool) {
	// Convert name to lowercase for case-insensitive comparison
	lowerName := make([]byte, len(name))
	for i, b := range name {
		if b >= 'A' && b <= 'Z' {
			lowerName[i] = b + 32 // Convert to lowercase
		} else {
			lowerName[i] = b
		}
	}

	// Search through stored headers
	for i := 0; i < req.headerCount; i++ {
		h := &req.headers[i]

		// Check if name length matches
		if h.NameLen != len(lowerName) {
			continue
		}

		// Compare header names (case-insensitive)
		match := true
		for j := 0; j < h.NameLen; j++ {
			if h.Name[j] != lowerName[j] {
				match = false
				break
			}
		}

		if match {
			return h.Value[:h.ValueLen], true
		}
	}

	return nil, false // Header not found
}

func (req *Request) AddCookie(cookie Cookie) {
	// if req.Headers["Set-Cookie"] == nil {
	// 	req.Headers["Set-Cookie"] = []string{}
	// }

	// req.Headers["Cookie"] = append(req.Headers["Cookie"], cookie.String())
}

func (req *Request) Cookie(name string) (Cookie, error) {
	// todo
	var cookie Cookie
	return cookie, nil

	// reqCookies, found := req.Headers["Cookie"]
	// if !found {
	// 	return cookie, ErrNoCookie
	// }

	// if err := cookie.Parse(data); err != nil {
	// 	return cookie, err
	// }
}

func (req *Request) urlDecode(src []byte, dst []byte) (int, error) {
	dstLen := 0
	for i := 0; i < len(src) && dstLen < len(dst); i++ {
		switch src[i] {
		case '%':
			if i+2 >= len(src) {
				return 0, errors.New("invalid URL encoding")
			}
			// Decode hex
			hi := hexToByte(src[i+1])
			lo := hexToByte(src[i+2])
			if hi == 255 || lo == 255 {
				return 0, errors.New("invalid hex in URL encoding")
			}
			dst[dstLen] = hi<<4 | lo
			i += 2
		case '+':
			dst[dstLen] = ' ' // + becomes space in query params
		default:
			dst[dstLen] = src[i]
		}
		dstLen++
	}
	return dstLen, nil
}
