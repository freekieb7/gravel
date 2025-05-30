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

	addr := "0.0.0.0:8080"
	server := http.NewServer("gravel")
	server.Router.Middleware = append(server.Router.Middleware, http.EnforceCookieMiddleware(), http.SessionMiddleware())

	server.Router.GET("/", func(request *http.Request, response http.Response) {
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

	server.Router.GET("/write_file", func(request *http.Request, response http.Response) {
		fs := filesystem.NewLocalFileSystem()
		fs.CreateFile("test.md")
	})

	server.Router.GET("/roll", func(request *http.Request, response http.Response) {
		ctx, span := tracer.Start(request.Context(), "roll")
		defer span.End()

		roll := 1 + rand.Intn(6)

		msg := "Anonymous player is rolling the dice"
		logger.InfoContext(ctx, msg, "result", roll)

		rollValueAttr := attribute.Int("roll.value", roll)
		span.SetAttributes(rollValueAttr)
		rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

		resp := strconv.Itoa(roll) + "\n"
		response.WithText(resp)
	})

	server.Router.GET("/ticker", func(request *http.Request, response http.Response) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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

		scheduler.Run(ctx)
	})

	server.Router.Group("/v1", func(group http.Router) {
		group.GET("/", func(request *http.Request, response http.Response) {
			response.WithJson(`{"test": "test"}`)
		}, exampleFirstMiddleware())
	}, exampleSecondMiddleware("my custom var"))

	serverErrorChannel := make(chan error, 1)
	go func() {
		log.Printf("Listening and serving on: %s", addr)
		serverErrorChannel <- server.ListenAndServe(ctx, addr)
	}()

	select {
	case err := <-serverErrorChannel:
		return err
	case <-ctx.Done():
		stop()
	}

	return server.Shutdown(ctx)
}

func exampleFirstMiddleware() http.MiddlewareFunc {
	return func(next http.Handler) http.HandleFunc {
		return func(request *http.Request, response http.Response) {
			log.Print("Executing middleware 1")
			next.ServeHTTP(request, response)
		}
	}
}

func exampleSecondMiddleware(myvar string) http.MiddlewareFunc {
	return func(next http.Handler) http.HandleFunc {
		return func(request *http.Request, response http.Response) {
			log.Printf("Executing middleware 2 : %s", myvar)
			next.ServeHTTP(request, response)
		}
	}
}
