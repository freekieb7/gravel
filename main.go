package main

import (
	"log"

	"github.com/freekieb7/gravel/http"
)

func main() {
	s := http.NewServer("hello", func(ctx *http.RequestCtx) {
		ctx.Response.WithText("hello world")
	})

	log.Fatal(s.ListenAndServe("0.0.0.0:8080"))

}
