module mygravel

go 1.24.3

require (
	github.com/freekieb7/gravel v0.0.0-20250530223757-cb73dc28076d
	go.opentelemetry.io/contrib/bridges/otelslog v0.11.0
	go.opentelemetry.io/otel v1.36.0
	go.opentelemetry.io/otel/metric v1.36.0
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/valyala/fasthttp v1.62.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/log v0.12.2 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/freekieb7/gravel => ../
