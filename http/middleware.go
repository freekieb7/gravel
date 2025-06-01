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

type MiddlewareFunc func(next HandleFunc) HandleFunc

func RecoverMiddleware() MiddlewareFunc {
	return func(next HandleFunc) HandleFunc {
		return func(ctx *RequestCtx) {
			defer func() {
				if recover := recover(); recover != nil {
					log.Println(recover)

					ctx.Response.WithText("something went wrong")
					return
				}
			}()

			next(ctx)
		}
	}
}

func EnforceCookieMiddleware() MiddlewareFunc {
	return func(next HandleFunc) HandleFunc {
		return func(ctx *RequestCtx) {
			_, err := ctx.Request.Cookie("SID")
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

				ctx.Request.AddCookie(cookie)
				ctx.Response.AddCookie(cookie)
			}

			next(ctx)
		}
	}
}

func SessionMiddleware() MiddlewareFunc {
	sessionStore := storage.NewMemorySessionStore()

	return func(next HandleFunc) HandleFunc {
		return func(ctx *RequestCtx) {
			cookie, err := ctx.Request.Cookie("SID")
			if errors.Is(err, http.ErrNoCookie) {
				ctx.Response.WithStatus(500)
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

			next(ctx)

			sessionStore.Save(sess)
		}
	}
}
