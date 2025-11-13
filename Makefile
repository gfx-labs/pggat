# This is the default target, which will be built when you invoke make
.PHONY: all

all: runotel

runotel: export OTEL_RESOURCE_ATTRIBUTES=deployment.environment=local,service.version=0.1.0,service.instance.id=$(HOSTNAME)
runotel: export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318/v1/traces
runotel:
	go run ./cmd/pggat run pool basic transaction

.PHONY: test
test:
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test

.PHONY: test-clean
test-clean:
	docker compose -f docker-compose.test.yml down -v

