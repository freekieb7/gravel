module example

go 1.24.3

require (
	github.com/freekieb7/gravel v0.0.0-20250722195223-7992bd7e5a90
	go.opentelemetry.io/contrib/bridges/otelslog v0.12.0
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/metric v1.37.0
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/log v0.13.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/freekieb7/gravel => ../../
