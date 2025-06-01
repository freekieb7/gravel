package http

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrNoCookie = errors.New("http: named cookie not present")

type Request struct {
	Method  string
	Path    string
	Headers Headers

	KeepAlive bool

	BodyRaw []byte
}

func (req *Request) AddCookie(cookie Cookie) {
	if req.Headers["Set-Cookie"] == nil {
		req.Headers["Set-Cookie"] = []string{}
	}

	req.Headers["Cookie"] = append(req.Headers["Cookie"], cookie.String())
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

	// return Cookie{
	// 	Name:        cookie.Name,
	// 	Value:       cookie.Value,
	// 	Path:        cookie.Path,
	// 	Domain:      cookie.Domain,
	// 	Expires:     cookie.Expires,
	// 	MaxAge:      cookie.MaxAge,
	// 	Secure:      cookie.Secure,
	// 	HttpOnly:    cookie.HttpOnly,
	// 	SameSite:    SameSite(cookie.SameSite),
	// 	Partitioned: cookie.Partitioned,
	// }, nil
}

func (req *Request) Read(reader *bufio.Reader) error {
	// Read request line
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			fmt.Println("Request read error:", err)
		}
		return err
	}
	requestLine = strings.TrimSpace(requestLine)
	if requestLine == "" {
		return io.EOF
	}
	parts := strings.Split(requestLine, " ")
	if len(parts) < 3 {
		return fmt.Errorf("malformed request line: %s", requestLine)
	}
	method, path, version := parts[0], parts[1], parts[2]

	req.Method = method
	req.Path = path

	// Read headers
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("header read error: %s", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // end of headers
		}
		if i := strings.Index(line, ":"); i >= 0 {
			key := strings.TrimSpace(line[:i])
			value := strings.TrimSpace(line[i+1:])
			headers[strings.ToLower(key)] = value
		}
	}

	// Determine keep-alive or not
	connHeader := strings.ToLower(headers["connection"])
	keepAlive := false
	if version == "HTTP/1.1" {
		keepAlive = connHeader != "close"
	} else if version == "HTTP/1.0" {
		keepAlive = connHeader == "keep-alive"
	}

	req.KeepAlive = keepAlive
	return nil
}

func (req *Request) Reset() {
	req.BodyRaw = nil
}
