package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/freekieb7/gravel/filesystem"
	"github.com/freekieb7/gravel/http"
	"github.com/freekieb7/gravel/scheduler"
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
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=gravel,service.namespace=freekieb7,deployment.environment=development")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:4317")
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
	if err := run(context.Background()); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	router := http.NewRouter()
	router.Middleware = append(router.Middleware, http.EnforceCookieMiddleware(), http.SessionMiddleware())

	router.GET("/", func(ctx *http.RequestCtx) {
		ctx.Response.WithJson("{\"test\": true}")
	})

	router.GET("/validation", func(ctx *http.RequestCtx) {
		violations := validation.ValidateMap(
			map[string]any{
				"title": []string{"testasdsadfsad"},
			},
			map[string][]string{
				"title": {"required", "max:255", "min:5"},
			},
		)

		if !violations.IsEmpty() {
			ctx.Response.WithJson(violations)
		} else {
			ctx.Response.WithText("ok")
		}
	})

	router.GET("/write_file", func(ctx *http.RequestCtx) {
		fs := filesystem.NewLocalFileSystem()
		fs.CreateFile("test.md")
	})

	router.GET("/roll", func(ctx *http.RequestCtx) {
		spanCtx, span := tracer.Start(context.Background(), "roll")
		defer span.End()

		roll := 1 + rand.Intn(6)

		msg := "Anonymous player is rolling the dice"
		logger.InfoContext(spanCtx, msg, "result", roll)

		rollValueAttr := attribute.Int("roll.value", roll)
		span.SetAttributes(rollValueAttr)
		rollCnt.Add(spanCtx, 1, metric.WithAttributes(rollValueAttr))

		resp := strconv.Itoa(roll) + "\n"
		ctx.Response.WithText(resp)
	})

	router.GET("/ticker", func(ctx *http.RequestCtx) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		task := scheduler.NewTask(
			func(number int) {
				fmt.Printf("Task %d: Time is now %s", number, time.Now())
			},
			0,
		)

		job := scheduler.NewJob().
			WithTasks(*task).
			WithInterval(time.Second * 2)

		scheduler := scheduler.NewScheduler()
		scheduler.AddJob(*job)

		scheduler.Run(timeoutCtx)
	})

	router.Group("/v1", func(group *http.Router) {
		group.GET("/v2", func(ctx *http.RequestCtx) {
			ctx.Response.WithJson(`{"test": "test"}`)
		}, exampleFirstMiddleware())
	}, exampleSecondMiddleware("my custom var"))

	serverErrorChannel := make(chan error, 1)

	addr := "0.0.0.0:8080"
	server := http.NewServer("gravel", router.Handler(), 2*8)

	go func() {
		log.Printf("Listening and serving on: %s", addr)
		serverErrorChannel <- server.ListenAndServe(addr)
	}()

	select {
	case err := <-serverErrorChannel:
		return err
	case <-ctx.Done():
		stop()
	}

	return server.Shutdown(ctx)
}

func exampleFirstMiddleware() http.Middleware {
	return func(next http.Handler) http.Handler {
		return func(ctx *http.RequestCtx) {
			log.Print("Executing middleware 1")
			next(ctx)
		}
	}
}

func exampleSecondMiddleware(myvar string) http.Middleware {
	return func(next http.Handler) http.Handler {
		return func(ctx *http.RequestCtx) {
			log.Printf("Executing middleware 2 : %s", myvar)
			next(ctx)
		}
	}
}
