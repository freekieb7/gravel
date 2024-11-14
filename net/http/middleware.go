package http

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/freekieb7/gravel/session"
	"github.com/freekieb7/gravel/session/storage"
)

type MiddlewareFunc func(next Handler) Handler

func RecoverMiddleware(next Handler) Handler {
	return HandlerFunc(func(request *Request, response *Response) {
		defer func() {
			if recover := recover(); recover != nil {
				log.Println(recover)

				response.WithText("something went wrong")
				return
			}
		}()

		next.ServeHTTP(request, response)
	})
}

func MethodCheckMiddleware(methods []string, next Handler) Handler {
	return HandlerFunc(func(request *Request, response *Response) {
		if !slices.Contains(methods, request.original.Method) {
			return
		}

		next.ServeHTTP(request, response)
	})
}

func EnforceCookieMiddleware(next Handler) Handler {
	return HandlerFunc(func(request *Request, response *Response) {
		_, err := request.Cookie("SID")
		if errors.Is(err, ErrNoCookie) {
			rawCookieValue := make([]byte, 16)
			rand.Read(rawCookieValue)

			cookie := &http.Cookie{
				Name:        "SID",
				Value:       base64.URLEncoding.EncodeToString(rawCookieValue),
				Expires:     time.Now().Add(365 * 24 * time.Hour),
				Secure:      true,
				HttpOnly:    true,
				Path:        "/",
				Partitioned: true,
				SameSite:    http.SameSiteStrictMode,
			}

			request.original.AddCookie(cookie) // Request
			http.SetCookie(response, cookie)   // Response
		}

		next.ServeHTTP(request, response)
	})
}

func SessionMiddleware(next Handler) Handler {
	sessionStore := storage.NewMemorySessionStore()

	return HandlerFunc(func(request *Request, response *Response) {
		cookie, err := request.Cookie("SID")
		if errors.Is(err, http.ErrNoCookie) {
			response.WithStatus(500)
			return
		}

		sessionId := cookie.Value()
		sess := session.NewDefaultSession(sessionId, "memses", make(map[string]any))

		if sessionStore.Has(sessionId) {
			attributes, err := sessionStore.Get(cookie.Value())
			if err != nil {
				log.Fatal(err)
			}

			sess.Replace(attributes)
		}

		next.ServeHTTP(request, response)

		sessionStore.Save(sess)
	})
}
