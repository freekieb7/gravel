package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"

	"github.com/freekieb7/gravel/http"
	"github.com/freekieb7/gravel/validation"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const name = "go.opentelemetry.io/otel/example/dice"

var (
	tracer  = otel.Tracer(name)
	meter   = otel.Meter(name)
	logger  = otelslog.NewLogger(name)
	rollCnt metric.Int64Counter
)

func init() {
	os.Setenv("OTEL_SERVICE_NAME", "gravel-test")
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=experimental,service.version=0.0.0")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://0.0.0.0:4317")
	os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")

	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls by roll value"),
		metric.WithUnit("{roll}"))
	if err != nil {
		panic(err)
	}

}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := http.NewServer("gravel")

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

	server.Router().Get("/roll", func(request http.Request, response http.Response) {
		ctx, span := tracer.Start(request.Context(), "roll")
		defer span.End()

		roll := 1 + rand.Intn(6)

		var msg string

		msg = "Anonymous player is rolling the dice"
		logger.InfoContext(ctx, msg, "result", roll)

		rollValueAttr := attribute.Int("roll.value", roll)
		span.SetAttributes(rollValueAttr)
		rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

		resp := strconv.Itoa(roll) + "\n"
		response.WithText(resp)
	})

	server.Router().Group("/v1", func(group http.Router) {
		group.Get("/", func(request http.Request, response http.Response) {
			response.WithJson(`{"test": "test"}`)
		}, exampleMiddleware)
	}, exampleMiddleware2)

	serverErrorChannel := make(chan error, 1)
	go func() {
		serverErrorChannel <- server.Run(ctx)
	}()

	// Wait for interruption.
	select {
	case err := <-serverErrorChannel:
		// Error when starting HTTP server.
		return err
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	return server.Shutdown(context.Background())
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
