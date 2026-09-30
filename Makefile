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

.PHONY: integration
integration:
	docker compose -f docker-compose.integration.yml up --build --abort-on-container-exit --exit-code-from integration-tests

.PHONY: integration-up
integration-up:
	docker compose -f docker-compose.integration.yml up -d postgres-primary postgres-replica pggat-transaction pggat-session pggat-hybrid

.PHONY: integration-down
integration-down:
	docker compose -f docker-compose.integration.yml down -v

.PHONY: integration-logs
integration-logs:
	docker compose -f docker-compose.integration.yml logs -f

.PHONY: integration-shell
integration-shell:
	docker compose -f docker-compose.integration.yml run --rm integration-tests sh

