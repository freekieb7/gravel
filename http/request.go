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

// Method implements Request.
func (request *Request) Method() string {
	return request.original.Method
}

func (request *Request) AddCookie(cookie Cookie) {
	request.original.AddCookie(
		&http.Cookie{
			Name:        cookie.Name,
			Value:       cookie.Value,
			Path:        cookie.Path,
			Domain:      cookie.Domain,
			Expires:     cookie.Expires,
			MaxAge:      cookie.MaxAge,
			Secure:      cookie.Secure,
			HttpOnly:    cookie.HttpOnly,
			SameSite:    http.SameSite(cookie.SameSite),
			Partitioned: cookie.Partitioned,
		},
	)
}

func (request *Request) Cookie(name string) (Cookie, error) {
	cookie, err := request.original.Cookie(name)

	if errors.Is(err, http.ErrNoCookie) {
		return Cookie{}, ErrNoCookie
	}

	return Cookie{
		Name:        cookie.Name,
		Value:       cookie.Value,
		Path:        cookie.Path,
		Domain:      cookie.Domain,
		Expires:     cookie.Expires,
		MaxAge:      cookie.MaxAge,
		Secure:      cookie.Secure,
		HttpOnly:    cookie.HttpOnly,
		SameSite:    SameSite(cookie.SameSite),
		Partitioned: cookie.Partitioned,
	}, nil
}

func (request *Request) Context() context.Context {
	return request.original.Context()
}
