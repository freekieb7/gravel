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

var (
	ErrNoCookie = errors.New("http: no cookie error")
)

type Middleware func(next Handler) Handler

func RecoverMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(req *Request, res *Response) {
			defer func() {
				if recover := recover(); recover != nil {
					log.Println(recover)

					res.WithText("something went wrong")
					return
				}
			}()

			next(req, res)
		}
	}
}

func EnforceCookieMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(req *Request, res *Response) {
			_, err := req.Cookie("SID")
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

				req.AddCookie(cookie)
				res.AddCookie(cookie)
			}

			next(req, res)
		}
	}
}

func SessionMiddleware() Middleware {
	sessionStore := storage.NewMemorySessionStore()

	return func(next Handler) Handler {
		return func(req *Request, res *Response) {
			cookie, err := req.Cookie("SID")
			if errors.Is(err, http.ErrNoCookie) {
				res.Status = StatusInternalServerError
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

			next(req, res)

			sessionStore.Save(sess)
		}
	}
}
