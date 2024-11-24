package main

import (
	"log"

	"github.com/freekieb7/gravel/http"
	"github.com/freekieb7/gravel/validation"
)

func main() {
	server := http.NewServer()

	server.Router().AddMiddleware(http.EnforceCookieMiddleware, http.SessionMiddleware)

	server.Router().Get("/", func(request http.Request, response http.Response) {
		violations := validation.ValidateMap(
			map[string]any{
				"title": []string{"test"},
			},
			map[string][]string{
				"title": {"required", "max:255", "min:5"},
			},
		)

		if !violations.IsEmpty() {
			response.WithJson(violations)
		} else {
			response.WithText("ok")
		}

	})

	server.Router().Group("/v1", func(group http.Router) {
		group.Get("/", func(request http.Request, response http.Response) {
			response.WithJson(`{"test": "test"}`)
		}, exampleMiddleware)
	}, exampleMiddleware2)

	server.Listen()
}

func exampleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(request http.Request, response http.Response) {
		log.Print("Executing middlewareOne")
		next.ServeHTTP(request, response)
	})
}

func exampleMiddleware2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(request http.Request, response http.Response) {
		log.Print("Executing middleware2")
		next.ServeHTTP(request, response)
	})
}
