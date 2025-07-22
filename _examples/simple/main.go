package main

import (
	"context"
	"encoding/json"
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

	router.GET("/", func(req *http.Request, res *http.Response) {
		res.WithJSON("{\"test\": true}")
	})

	router.GET("/params", func(req *http.Request, res *http.Response) {
		qTest, found := req.QueryParam([]byte("test"))
		if found {
			res.WithText(string(qTest))
		} else {
			res.WithText("not found")
		}
	})

	router.POST("/post", func(req *http.Request, res *http.Response) {
		var m map[string]any
		if err := json.Unmarshal(req.Body, &m); err != nil {
			res.WithText("bad json")
			return
		}

		for k, v := range m {
			log.Printf("key: %s, value: %v\n", k, v)
		}
	})

	router.GET("/validation", func(req *http.Request, res *http.Response) {
		violations := validation.ValidateMap(
			map[string]any{
				"title": []string{"testasdsadfsad"},
			},
			map[string][]string{
				"title": {"required", "max:255", "min:5"},
			},
		)

		if !violations.IsEmpty() {
			res.WithJSON(violations)
		} else {
			res.WithText("ok")
		}
	})

	router.GET("/write_file", func(req *http.Request, res *http.Response) {
		fs := filesystem.NewLocalFileSystem()
		fs.CreateFile("test.md")
	})

	router.GET("/roll", func(req *http.Request, res *http.Response) {
		spanCtx, span := tracer.Start(context.Background(), "roll")
		defer span.End()

		roll := 1 + rand.Intn(6)

		msg := "Anonymous player is rolling the dice"
		logger.InfoContext(spanCtx, msg, "result", roll)

		rollValueAttr := attribute.Int("roll.value", roll)
		span.SetAttributes(rollValueAttr)
		rollCnt.Add(spanCtx, 1, metric.WithAttributes(rollValueAttr))

		resp := strconv.Itoa(roll) + "\n"
		res.WithText(resp)
	})

	router.GET("/ticker", func(req *http.Request, res *http.Response) {
		go func() {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			task := scheduler.NewTask(
				func(number int) {
					fmt.Printf("Task %d: Time is now %s\n", number, time.Now())
				},
				0,
			)

			job := scheduler.NewJob().
				WithTasks(*task).
				WithInterval(time.Second * 2)

			scheduler := scheduler.NewScheduler()
			scheduler.AddJob(*job)

			scheduler.Run(timeoutCtx)
		}()
	})

	router.Group("/v1", func(group *http.Router) {
		group.GET("/v2", func(req *http.Request, res *http.Response) {
			res.WithJSON(`{"test": "test"}`)
		}, exampleFirstMiddleware())
	}, exampleSecondMiddleware("my custom var"))

	serverErrorChannel := make(chan error, 1)

	addr := "0.0.0.0:8080"
	server := http.NewServer(router.Handler())

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
		return func(req *http.Request, res *http.Response) {
			log.Print("Executing middleware 1")
			next(req, res)
		}
	}
}

func exampleSecondMiddleware(myvar string) http.Middleware {
	return func(next http.Handler) http.Handler {
		return func(req *http.Request, res *http.Response) {
			log.Printf("Executing middleware 2 : %s", myvar)
			next(req, res)
		}
	}
}
