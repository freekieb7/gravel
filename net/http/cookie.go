package http

import "net/http"

type Cookie struct {
	original http.Cookie
}

func (cookie Cookie) Value() string {
	return cookie.original.Value
}
