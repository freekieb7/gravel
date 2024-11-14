package main

import (
	"log"

	"github.com/freekieb7/gravel/net/http"
)

func main() {
	server := http.NewServer()

	server.Router.Add(http.EnforceCookieMiddleware, http.SessionMiddleware)

	server.Router.Get("/", func(request *http.Request, response *http.Response) {
		response.Write([]byte("test"))
	})

	server.Router.Group("/v1", func(group *http.Router) {
		group.Get("/", func(request *http.Request, response *http.Response) {
			response.WithJson(`{"test": "test"}`)
		}, exampleMiddleware)
	}, exampleMiddleware2)

	server.Listen()
}

func exampleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(request *http.Request, response *http.Response) {
		log.Print("Executing middlewareOne")
		next.ServeHTTP(request, response)
	})
}

func exampleMiddleware2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(request *http.Request, response *http.Response) {
		log.Print("Executing middleware2")
		next.ServeHTTP(request, response)
	})
}
