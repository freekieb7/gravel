package http

import (
	"bufio"
	"encoding/json"
	"fmt"
)

type Headers map[string][]string

type Response struct {
	Status  Status
	Headers Headers
	Body    []byte
}

func (res *Response) AddCookie(cookie Cookie) {
	if res.Headers["Set-Cookie"] == nil {
		res.Headers["Set-Cookie"] = []string{}
	}

	res.Headers["Set-Cookie"] = append(res.Headers["Set-Cookie"], cookie.String())
}

func (res *Response) WithStatus(status Status) *Response {
	res.Status = status
	return res
}

func (res *Response) WithJson(payload any) *Response {
	res.Headers["Content-Type"] = []string{"application/json"}

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
	res.Headers["Content-Type"] = []string{"text/plain"}
	res.Body = []byte(payload)
	return res
}

func (res *Response) Write(bw *bufio.Writer) error {
	// Start line
	bw.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", res.Status, res.Status.AsText()))

	// Headers
	for name, header := range res.Headers {
		for _, value := range header {
			bw.WriteString(fmt.Sprintf("%s: %s\r\n", name, value))
		}
	}
	bw.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(res.Body)))

	// Empty line
	bw.WriteString("\r\n")

	// Body
	bw.Write(res.Body)

	// End
	return bw.Flush()
}

func (res *Response) Reset() {
	res.Status = 200
	res.Body = make([]byte, 0)
	res.Headers = Headers{}
}
