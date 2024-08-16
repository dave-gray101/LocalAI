package startup

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Takes a pointer to a context if one should be used
// Otherwise, pass nil and it will be initialized to use context.Background()
// TODO: Determine if there's value in ever using anything but context.Background()
// Returns a safe shutdown function
func InitializeMetricsProviders(ctx *context.Context) (func() error, error) {
	if ctx == nil {
		bkg := context.Background() // TODO: check why this local is needed?
		ctx = &bkg
	}

	res, err := resource.New(*ctx,
		resource.WithAttributes(
			semconv.ServiceName("LocalAI"),
		),
	)
	if err != nil {
		return nil, err
	}

	traceExporter, err := otlptracehttp.New(*ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint("jaeger:4317"),
	)
	if err != nil {
		return nil, fmt.Errorf("error during jaeger trace exporter setup: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(*ctx,
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithEndpoint("prometheus:9090"),
	)
	if err != nil {
		return nil, fmt.Errorf("error during prometheus meter exporter setup: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	return func() error {
		var err error
		if err = tracerProvider.Shutdown(*ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}

		if err2 := meterProvider.Shutdown(*ctx); err2 != nil {
			log.Printf("Error shutting down meter provider: %v", err)
			err = errors.Join(err, err2)
		}

		return err
	}, nil
}
