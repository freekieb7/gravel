package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
)

type Headers map[string][]string

type Response struct {
	Status          uint16
	HeaderNameList  [MaxResponseHeaders][]byte
	HeaderValueList [MaxResponseHeaders][]byte
	Body            []byte

	statusBuf [3]byte
	lenBuf    [20]byte
}

func (res *Response) Reset() {
	res.Status = 200
	// res.HeaderNameList = [MaxResponseHeaders][]byte{}
	res.Body = nil
}

func (res *Response) AddCookie(cookie Cookie) {
	res.SetHeader([]byte("Set-Cookie"), []byte(cookie.String()))
}

func (res *Response) WithStatus(status uint16) *Response {
	res.Status = status
	return res
}

func (res *Response) WithJson(payload any) *Response {
	res.SetHeader([]byte("Content-Type"), []byte("application/json"))

	switch p := payload.(type) {
	case string:
		{
			res.Body = []byte(p)
		}
	case []byte:
		{
			res.Body = p
		}
	default:
		{
			data, _ := json.Marshal(p)
			res.Body = data
		}
	}

	return res
}

func (res *Response) WithText(payload string) *Response {
	res.SetHeader([]byte("Content-Type"), []byte(payload))
	res.Body = []byte(payload)
	return res
}

func (res *Response) SetHeader(key, value []byte) {
	for i, headerName := range res.HeaderNameList {
		if len(headerName) == 0 || bytes.EqualFold(headerName, key) {
			res.HeaderNameList[i] = key
			res.HeaderValueList[i] = value
			return
		}
	}
}

func (res *Response) AddHeader(key, value []byte) {
	for i, headerName := range res.HeaderNameList {
		if len(headerName) == 0 {
			res.HeaderNameList[i] = key
			res.HeaderValueList[i] = value
			return
		}

		if bytes.EqualFold(headerName, key) {
			res.HeaderValueList[i] = append(res.HeaderValueList[i], ';')
			res.HeaderValueList[i] = append(res.HeaderValueList[i], value...)
			break
		}
	}
}

func (res *Response) Write(bw *bufio.Writer) error {
	// Start line
	bw.WriteString("HTTP/1.1 ")
	statusStr := strconv.AppendInt(res.statusBuf[:0], int64(res.Status), 10)
	bw.Write(statusStr)
	bw.WriteByte(' ')
	bw.WriteString(statusMessages[res.Status])
	bw.WriteString("\r\n")

	// Headers
	for i, headerName := range res.HeaderNameList {
		if len(headerName) == 0 {
			break
		}
		bw.Write(headerName)
		bw.WriteString(": ")
		bw.Write(res.HeaderValueList[i])
		bw.WriteString("\r\n")
	}

	bw.WriteString("Content-Length: ")
	lenStr := strconv.AppendInt(res.lenBuf[:0], int64(len(res.Body)), 10)
	bw.Write(lenStr)
	bw.WriteString("\r\n\r\n")

	// Body
	bw.Write(res.Body)

	return bw.Flush()
}
