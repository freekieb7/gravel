package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
)

type Response struct {
	Status          uint16
	HeaderNameList  [MaxResponseHeaders][64]byte
	HeaderNameLens  [MaxResponseHeaders]int
	HeaderValueList [MaxResponseHeaders][]byte
	HeaderCount     int
	Body            []byte

	statusBuf [3]byte
	lenBuf    [20]byte
}

func (res *Response) Reset() {
	for i := 0; i < res.HeaderCount; i++ {
		res.HeaderNameLens[i] = 0
		res.HeaderValueList[i] = nil
	}
	res.HeaderCount = 0
	res.Body = nil
	res.Status = 200
}

func (res *Response) AddCookie(cookie Cookie) {
	res.SetHeader([]byte("set-cookie"), []byte(cookie.String()))
}

func (res *Response) WithStatus(status uint16) *Response {
	res.Status = status
	return res
}

func (res *Response) WithJson(payload any) *Response {
	res.SetHeader([]byte("content-type"), []byte("application/json"))

	switch p := payload.(type) {
	case string:
		res.Body = []byte(p)
	case []byte:
		res.Body = p
	default:
		data, _ := json.Marshal(p)
		res.Body = data
	}

	return res
}

func (res *Response) WithText(payload string) *Response {
	res.SetHeader([]byte("content-type"), []byte(payload))
	res.Body = []byte(payload)
	return res
}

// Optimized SetHeader: lower-case key, scan only up to HeaderCount, no EqualFold
func (res *Response) SetHeader(key, value []byte) {
	n := len(key)
	if n > 64 {
		n = 64
	}
	var lowerKey [64]byte
	for i := 0; i < n; i++ {
		c := key[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lowerKey[i] = c
	}
	lookup := lowerKey[:n]

	for i := 0; i < res.HeaderCount; i++ {
		headerName := res.HeaderNameList[i][:res.HeaderNameLens[i]]
		if len(headerName) != n {
			continue
		}
		eq := true
		for j := 0; j < n; j++ {
			if headerName[j] != lookup[j] {
				eq = false
				break
			}
		}
		if eq {
			res.HeaderValueList[i] = value
			return
		}
	}
	if res.HeaderCount < MaxResponseHeaders {
		copy(res.HeaderNameList[res.HeaderCount][:], lookup)
		res.HeaderNameLens[res.HeaderCount] = n
		res.HeaderValueList[res.HeaderCount] = value
		res.HeaderCount++
	}
}

func (res *Response) AddHeader(key, value []byte) {
	for i, headerName := range res.HeaderNameList {
		if len(headerName) == 0 {
			n := len(key)
			if n > 64 {
				n = 64
			}
			copy(res.HeaderNameList[i][:], key[:n])
			res.HeaderNameLens[i] = n
			res.HeaderValueList[i] = value
			return
		}
		if bytes.EqualFold(headerName[:res.HeaderNameLens[i]], key) {
			res.HeaderValueList[i] = append(res.HeaderValueList[i], ';')
			res.HeaderValueList[i] = append(res.HeaderValueList[i], value...)
			break
		}
	}
}

func (res *Response) Write(bw *bufio.Writer) error {
	// Write status line
	bw.WriteString("HTTP/1.1 ")
	statusStr := strconv.AppendInt(res.statusBuf[:0], int64(res.Status), 10)
	bw.Write(statusStr)
	bw.WriteByte(' ')
	bw.WriteString(statusMessages[res.Status])
	bw.WriteString("\r\n")

	// Write headers in a tight loop
	for i := 0; i < res.HeaderCount; i++ {
		headerName := res.HeaderNameList[i][:res.HeaderNameLens[i]]
		bw.Write(headerName)
		bw.WriteString(": ")
		bw.Write(res.HeaderValueList[i])
		bw.WriteString("\r\n")
	}

	// Write Content-Length header
	bw.WriteString("content-length: ")
	lenStr := strconv.AppendInt(res.lenBuf[:0], int64(len(res.Body)), 10)
	bw.Write(lenStr)
	bw.WriteString("\r\n\r\n")

	// Write body directly
	if len(res.Body) > 0 {
		bw.Write(res.Body)
	}

	return bw.Flush()
}
