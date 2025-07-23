package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"

	"github.com/freekieb7/gravel/http"
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
	os.Setenv("OTEL_SERVICE_NAME", "gravel-otel-example")
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.namespace=mynamespace,deployment.environment=development")
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

	serverErrCh := make(chan error, 1)

	addr := "0.0.0.0:8080"
	server := http.NewServer(router.Handler())

	go func() {
		log.Printf("Listening and serving on: %s", addr)
		serverErrCh <- server.ListenAndServe(addr)
	}()

	select {
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
		stop()
	}

	return server.Shutdown(ctx)
}
