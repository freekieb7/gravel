.DEFAULT_GOAL := help

help:
	@echo "Available targets:"
	@echo "  lint      - Run go vet and golangci-lint"
	@echo "  test      - Run go tests"
	@echo "  fmt       - Run go fmt"
	@echo "  modtidy   - Run go mod tidy"
	@echo "  clean     - Clean build artifacts and profiles"
	@echo "  deps      - Download go module dependencies"

.PHONY: lint
lint:
	go vet ./...
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v2.3.0 golangci-lint run ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: modtidy
modtidy:
	go mod tidy

.PHONY: clean
clean:
	go clean
	rm -f http/cpu.prof http/mem.prof

.PHONY: deps
deps:
	go mod download

# pprof:
# 	go test -test.benchmem -cpuprofile cpu.prof -memprofile mem.prof -bench BenchmarkServerGet ./http/...
# 	go tool pprof -http localhost:8080 http/cpu.prof

# bench:
# 	GOMAXPROCS=4 go test -bench=kServerGet -benchmem -benchtime=10s ./http/...