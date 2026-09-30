# This is the default target, which will be built when you invoke make
.PHONY: all

all: runotel

runotel: export OTEL_RESOURCE_ATTRIBUTES=deployment.environment=local,service.version=0.1.0,service.instance.id=$(HOSTNAME)
runotel: export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318/v1/traces
runotel:
	go run ./cmd/pggat run pool basic transaction

GOLANGCI_LINT_VERSION ?= v2.14.0

.PHONY: test
test:
	go test -race ./...

.PHONY: integration
integration:
	go test -race -tags integration ./test/integration/...

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --build-tags integration ./test/integration/...
