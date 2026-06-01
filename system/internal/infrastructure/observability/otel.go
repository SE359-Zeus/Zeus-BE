package observability

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracer initializes an OTLP/HTTP exporter and registers it as the global
// tracer provider. If otlpEndpoint is empty, returns (nil, nil).
func InitTracer(ctx context.Context, serviceName, env, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	if otlpEndpoint == "" {
		return nil, nil
	}

	// Normalize endpoint: accept full URL or host:port; otlptracehttp wants host:port
	ep := otlpEndpoint
	insecure := false
	if strings.HasPrefix(ep, "http://") {
		insecure = true
		ep = strings.TrimPrefix(ep, "http://")
	} else if strings.HasPrefix(ep, "https://") {
		ep = strings.TrimPrefix(ep, "https://")
	}
	if idx := strings.Index(ep, "/"); idx != -1 {
		ep = ep[:idx]
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(ep)}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)

	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
		"",
		attribute.String("service.name", serviceName),
		attribute.String("env", env),
	))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}
