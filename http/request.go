package http

import (
	"context"
	"errors"
	"net/http"
)

var ErrNoCookie = errors.New("http: named cookie not present")

type Request interface {
	Method() string
	Cookie(name string) (Cookie, error)
	AddCookie(cookie Cookie)
	Context() context.Context
}

type request struct {
	original *http.Request
}

// Method implements Request.
func (request *request) Method() string {
	return request.original.Method
}

func (request *request) AddCookie(cookie Cookie) {
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

func (request *request) Cookie(name string) (Cookie, error) {
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

func (request *request) Context() context.Context {
	return request.original.Context()
}
