package http

import (
	"encoding/json"
	"log"
	"net/http"
)

type Response struct {
	original http.ResponseWriter
}

func (response *Response) AddCookie(cookie Cookie) {
	http.SetCookie(response.original, &http.Cookie{
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
	})
}

func (response *Response) WithStatus(status int) *Response {
	response.original.WriteHeader(status)
	return response
}

func (response *Response) WithJson(payload any) *Response {
	response.original.Header().Set("Content-Type", "application/json")

	if vStr, ok := payload.(string); ok {
		response.original.Write([]byte(vStr))
	} else if err := json.NewEncoder(response.original).Encode(payload); err != nil {
		log.Fatalf("response: encoding data to json failed")
	}

	return response
}

func (response *Response) WithText(payload string) *Response {
	response.original.Header().Set("Content-Type", "text/plain")
	response.original.Write([]byte(payload))
	return response
}
