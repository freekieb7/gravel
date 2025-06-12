package http

import (
	"bufio"
	"bytes"
	"errors"
)

var ErrNoCookie = errors.New("http: named cookie not present")

type Request struct {
	Method   []byte
	Path     []byte
	Protocol []byte

	HeaderNameList  [MaxRequestHeaders][]byte
	HeaderValueList [MaxRequestHeaders][]byte

	BodyRaw []byte
}

func (req *Request) HeaderValue(key string) ([]byte, bool) {
headerLoop:
	for i, headerName := range req.HeaderNameList {
		if len(headerName) != len(key) {
			continue
		}

		for i, kc := range key {
			hc := headerName[i]

			if kc == rune(hc) {
				continue
			}

			// try with key as lower case
			if kc >= 'A' && kc <= 'Z' && kc+0x20 == rune(hc) {
				continue
			}

			// try with key as upper case
			if kc >= 'a' && kc <= 'z' && kc-0x20 == rune(hc) {
				continue
			}

			continue headerLoop
		}

		return req.HeaderValueList[i], true
	}

	return nil, false
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

func (req *Request) Parse(br *bufio.Reader) error {
	b, _, err := br.ReadLine()
	if err != nil {
		return err
	}

	// parse method
	n := bytes.IndexByte(b, ' ')
	if n < 0 {
		return errors.New("cannot find http request method")
	}
	req.Method = b[:n]
	b = b[n+1:]

	// parse path
	n = bytes.LastIndexByte(b, ' ')
	if n < 0 {
		return errors.New("cannot find http request path")
	}
	req.Path = b[:n]
	b = b[n+1:]

	// parse protocol
	req.Protocol = b[:]

	// // parse status
	// var status uint16
	// for i := 0; i < 3; i++ {
	// 	status |= uint16(int(b[i]-'0') * (i * 10))
	// }

	// Read request line
	// requestLine, err := reader.ReadString('\n')
	// if err != nil {
	// 	if err != io.EOF {
	// 		fmt.Println("Request read error:", err)
	// 	}
	// 	return err
	// }
	// requestLine = strings.TrimSpace(requestLine)
	// if requestLine == "" {
	// 	return io.EOF
	// }
	// parts := strings.Split(requestLine, " ")
	// if len(parts) < 3 {
	// 	return fmt.Errorf("malformed request line: %s", requestLine)
	// }
	// method, path, version := parts[0], parts[1], parts[2]

	// req.Method = method
	// req.Path = path

	// // Read headers
	for i := range MaxRequestHeaders {
		b, _, err := br.ReadLine()
		if err != nil {
			return err
		}
		if len(b) == 0 {
			break
		}

		cn := bytes.IndexByte(b, ' ')
		if cn < 0 {
			return errors.New("cannot find http request header name")
		}
		req.HeaderNameList[i] = b[:cn-1] // ignore colon

		req.HeaderValueList[i] = b[cn+1:]
	}

	return nil
}
