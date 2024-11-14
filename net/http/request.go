package http

import (
	"context"
	"errors"
	"net/http"
)

var ErrNoCookie = errors.New("http: named cookie not present")

type Request struct {
	original *http.Request
}

func (request *Request) Cookie(name string) (Cookie, error) {
	originalCookie, err := request.original.Cookie(name)

	if errors.Is(err, http.ErrNoCookie) {
		return Cookie{}, ErrNoCookie
	}

	return Cookie{*originalCookie}, nil
}

func (request *Request) Context() context.Context {
	return request.original.Context()
}
