package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

// Structure for Open Telemetry variables
type Telemetry struct {
	server   *http.Server          // If type of metrics collection == "scraper".
	Provider *metric.MeterProvider // If not scraper use gRPC.
	meter    api.Meter             // meter to create metrics.
	ctx      *context.Context
}

var (
	once sync.Once
)

// Initialize metrics depending on the configuration parameter value.
func (t *Telemetry) InitMetrics(meterName string, ctx *context.Context) *Telemetry {
	metricsExporter := getEnvWithDefault("METRICS_EXPORTER", "")
	t.ctx = ctx

	once.Do(func() {
		if metricsExporter == "scraper" {
			slog.Info("Starting metrics with scraper exporter")
			t.initScrapeMetrics(meterName) // Serves a page on http://localhost:9080/metrics .
		} else {
			slog.Info("Starting metrics with grpc exporter")
			t.initGRPCMetrics(meterName) // Sends data to localhost:4317 or whatever OTEL_EXPORTER_OTLP_METRICS_ENDPOINT is set to.
		}
	})
	return &Telemetry{
		server:   t.server,
		Provider: t.Provider,
		meter:    t.meter,
		ctx:      t.ctx,
	}
}

// getEnvWithDefault gets an environment variable with a default fallback
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (t *Telemetry) Close() {
	if t.Provider != nil {
		t.Provider.ForceFlush(*t.ctx)
	}
}

// Initialize GRPC metrics exporter. https://opentelemetry.io/docs/languages/go/exporters/#otlp-metrics-over-grpc.
func (t *Telemetry) initGRPCMetrics(meterName string) {
	// The URL to export is set via environment variable
	// OTEL_EXPORTER_OTLP_METRICS_ENDPOINT and if not set it is  "localhost:4317"
	exporter, err := otlpmetricgrpc.New(*t.ctx)
	if err != nil {
		slog.Error("Creating GRPC exporter", "error", err)

		return
	}

	t.Provider = metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exporter)))
	otel.SetMeterProvider(t.Provider)
	t.meter = t.Provider.Meter(meterName)
}

// Ititialize scrape metrics exporter. https://github.com/open-telemetry/opentelemetry-go/blob/main/example/prometheus/main.go.
func (t *Telemetry) initScrapeMetrics(meterName string) {
	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := prometheus.New()
	if err != nil {
		slog.Error("Creating HTML scrape exporter", "error", err)

		return
	}

	t.Provider = metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(t.Provider)
	t.meter = t.Provider.Meter(meterName)

	go t.serveMetrics()
}

// Run metrics server for "scraper" open telemetry collector
func (t *Telemetry) serveMetrics() {
	slog.Info("Serving metrics at localhost:9080/metrics")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	t.server = &http.Server{
		Addr:    ":9080",
		Handler: mux,
	}

	err := t.server.ListenAndServe()
	if err != nil {
		if fmt.Sprint(err) == "http: Server closed" {
			slog.Info("Shutting down server", "message", err)
		} else {
			slog.Error("ListenAndServe exited with", "error", err)
		}

		return
	}
}

// Shutdown HTTP server used for "scraper" metrics collection.
func (t *Telemetry) shutdownScraperMetrics() {
	if t.server != nil {
		_ = t.server.Shutdown(*t.ctx)
		slog.Info("Shutting down metrics server")
	}
}
