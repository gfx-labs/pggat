# This is the default target, which will be built when you invoke make
.PHONY: all

all: runotel

devenv:
	export GFX_CORE_ALLOCATION=0

otelenv:
	export OTEL_RESOURCE_ATTRIBUTES=deployment.environment=local,service.version=0.1.0,service.instance.id=$(HOSTNAME)
	export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318/v1/traces

runotel: devenv otelenv
	go run ./cmd/pggat run pool basic transaction

