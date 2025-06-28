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

	lowerKey [64]byte // adjust size as needed

}

func (req *Request) Reset() {
}

func (req *Request) HeaderValue(key []byte) ([]byte, bool) {
	if len(key) == 0 {
		return nil, false
	}
	if len(key) > len(req.lowerKey) {
		return nil, false // key too long
	}
	for i := range key {
		c := key[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		req.lowerKey[i] = c
	}
	lookup := req.lowerKey[:len(key)]

	// Use HeaderCount if you add it, else keep nil check
	for i := 0; i < MaxRequestHeaders; i++ {
		headerName := req.HeaderNameList[i]
		if headerName == nil {
			break
		}
		if len(headerName) != len(lookup) {
			continue
		}
		// Inline comparison
		eq := true
		for j := range lookup {
			if headerName[j] != lookup[j] {
				eq = false
				break
			}
		}
		if eq {
			return req.HeaderValueList[i], true
		}
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

	// // Read headers
	for i := range MaxRequestHeaders {
		b, _, err := br.ReadLine()
		if err != nil {
			return err
		}
		if len(b) == 0 {
			if i+1 < MaxRequestHeaders {
				req.HeaderNameList[i] = nil
				req.HeaderValueList[i] = nil
			}
			break
		}

		cn := bytes.IndexByte(b, ' ')
		if cn < 0 {
			return errors.New("cannot find http request header name")
		}
		// Lower-case header name in-place (ASCII only, zero alloc)
		name := b[:cn-1]
		for j := range name {
			if name[j] >= 'A' && name[j] <= 'Z' {
				name[j] += 'a' - 'A'
			}
		}
		req.HeaderNameList[i] = name
		req.HeaderValueList[i] = b[cn+1:]
	}

	return nil
}
