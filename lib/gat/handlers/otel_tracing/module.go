package otel_tracing

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/util/go/gotel"
	"github.com/caddyserver/caddy/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

const ModuleID = "pggat.handlers.tracing.otel"

type Config struct {
	ServiceName      string         `json:"service_name,omitempty"`
	ServiceNamespace string         `json:"service_namespace,omitempty"`
	Endpoint         string         `json:"endpoint,omitempty"`
	BatchTimout      *time.Duration `json:"batch_timout,omitempty"`
	LogLevel         slog.Level     `json:"log_level,omitempty"`
	SamplerRate      string         `json:"sample_rate,omitempty"`
}

func defaultConfig() Config {
	c := Config{
		ServiceName:      "pggat",
		ServiceNamespace: "gfx.cafe/gfx",
		LogLevel:         slog.LevelInfo,
	}

	return c
}

func NewModule() *Module {
	return &Module{
		Config: defaultConfig(),
	}
}

type Module struct {
	Config

	tracer     trace.Tracer
	shutdownFn gotel.ShutdownFunc
	logger     *zap.Logger
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: ModuleID,
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	T.logger = ctx.Logger(T)

	T.tracer = otel.Tracer("pggat", trace.WithInstrumentationAttributes(
		attribute.String("component", "gfx.cafe/gfx/pggat/lib/gat/handlers/otel_tracing/module.go"),
	))

	providerOptions := []gotel.Option{
		gotel.WithServiceName(T.Config.ServiceName),
		gotel.WithServiceNamespace(T.Config.ServiceNamespace),
	}

	if T.Config.BatchTimout != nil {
		providerOptions = append(providerOptions, gotel.WithBatchTimeout(*T.Config.BatchTimout))
	}

	if T.Config.Endpoint != "" {
		providerOptions = append(providerOptions, gotel.WithEndpoint(T.Config.Endpoint))
	}

	if T.Config.SamplerRate != "" {
		sampler, err := mapSamplerType(T.Config.SamplerRate)
		if err != nil {
			return err
		}
		providerOptions = append(providerOptions, gotel.WithSampler(sampler))
	}

	var err error
	T.shutdownFn, err = gotel.InitTracing(ctx.Context, providerOptions...)
	return err
}

func (T *Module) Handle(next gat.Router) gat.Router {
	return gat.RouterFunc(func(ctx context.Context, c *fed.Conn) error {
		ctx, span := T.tracer.Start(ctx, "route handler",
			trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		return next.Route(ctx, c)
	})
}

func (T *Module) Cancel(context.Context,fed.BackendKey) {}

func (T *Module) Cleanup() (err error) {
	if T.shutdownFn != nil {
		err = T.shutdownFn(context.Background())
	}
	return
}

func mapSamplerType(samplerType string) (sampler sdktrace.Sampler, err error) {
	switch strings.ToLower(samplerType) {
	case "never", "none", "off":
		sampler = sdktrace.NeverSample()
	case "always", "all", "on":
		sampler = sdktrace.AlwaysSample()
	default:
		var val float64
		if val, err = strconv.ParseFloat(samplerType, 64); err == nil {
			// if not 0.0 -> 1.0, then assume that the representation is a % (0-100)
			if val > 1 {
				val = val / float64(100)
			}
			if val >= 0.0 && val <= 1.0 {
				sampler = sdktrace.TraceIDRatioBased(val)
			} else {
				err = fmt.Errorf("sampler ratio must be >= 0.0 and <= 1.0: '%s'", sampler)
			}
		} else {
			err = fmt.Errorf("unknown sampler type/ratio value: '%s': %v", sampler, err)
		}
	}

	return
}

var _ gat.Handler = (*Module)(nil)
var _ gat.CancellableHandler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
var _ caddy.CleanerUpper = (*Module)(nil)
