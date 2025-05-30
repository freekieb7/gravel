package http

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/freekieb7/gravel/session"
	"github.com/freekieb7/gravel/session/storage"
)

type MiddlewareFunc func(next Handler) HandleFunc

func RecoverMiddleware() MiddlewareFunc {
	return func(next Handler) HandleFunc {
		return func(request *Request, response Response) {
			defer func() {
				if recover := recover(); recover != nil {
					log.Println(recover)

					response.WithText("something went wrong")
					return
				}
			}()

			next.ServeHTTP(request, response)
		}
	}
}

func EnforceCookieMiddleware() MiddlewareFunc {
	return func(next Handler) HandleFunc {
		return func(request *Request, response Response) {
			_, err := request.Cookie("SID")
			if errors.Is(err, ErrNoCookie) {
				rawCookieValue := make([]byte, 16)
				rand.Read(rawCookieValue)

				cookie := Cookie{
					Name:        "SID",
					Value:       base64.URLEncoding.EncodeToString(rawCookieValue),
					Expires:     time.Now().Add(365 * 24 * time.Hour),
					Secure:      true,
					HttpOnly:    true,
					Path:        "/",
					Partitioned: true,
					SameSite:    SameSiteStrictMode,
				}

				request.AddCookie(cookie)
				response.AddCookie(cookie)
			}

			next.ServeHTTP(request, response)
		}
	}
}

func SessionMiddleware() MiddlewareFunc {
	sessionStore := storage.NewMemorySessionStore()

	return func(next Handler) HandleFunc {
		return func(request *Request, response Response) {
			cookie, err := request.Cookie("SID")
			if errors.Is(err, http.ErrNoCookie) {
				response.WithStatus(500)
				return
			}

			sessionId := cookie.Value
			sess := session.NewDefaultSession(sessionId, "memses", make(map[string]any))

			if sessionStore.Has(sessionId) {
				attributes, err := sessionStore.Get(sessionId)
				if err != nil {
					log.Fatal(err)
				}

				sess.Replace(attributes)
			}

			next.ServeHTTP(request, response)

			sessionStore.Save(sess)
		}
	}
}
