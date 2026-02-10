package metrics

import (
	"context"
	_ "embed"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for gRPC mock server
type Metrics struct {
	// gRPC metrics
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ErrorsTotal     *prometheus.CounterVec
	PanicsTotal     *prometheus.CounterVec

	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Resource metrics (custom gauges)
	MemoryUsage *prometheus.GaugeVec
	Goroutines  prometheus.Gauge

	registry *prometheus.Registry
	server   *http.Server
	port     string
}

// New creates a new Metrics instance with all metrics registered
func New(port string) *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_request_duration_seconds",
				Help:    "Histogram of gRPC request latencies",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "status"},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_errors_total",
				Help: "Total number of gRPC errors by error message",
			},
			[]string{"method", "error"},
		),
		PanicsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_panics_total",
				Help: "Total number of gRPC panics by panic message",
			},
			[]string{"method", "panic"},
		),
		MemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "grpc_memory_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"type"},
		),
		Goroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "grpc_goroutines_total",
				Help: "Number of goroutines",
			},
		),
		registry: registry,
		port:     port,
	}

	// Register gRPC metrics
	registry.MustRegister(m.RequestsTotal)
	registry.MustRegister(m.RequestDuration)
	registry.MustRegister(m.ErrorsTotal)
	registry.MustRegister(m.PanicsTotal)
	registry.MustRegister(m.MemoryUsage)
	registry.MustRegister(m.Goroutines)

	// Register default Go collectors for CPU, memory, GC stats
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return m
}

// NewHTTP creates a new Metrics instance for HTTP server with HTTP-specific metrics
func NewHTTP(port string) *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Histogram of HTTP request latencies",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		MemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_memory_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"type"},
		),
		Goroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_goroutines_total",
				Help: "Number of goroutines",
			},
		),
		registry: registry,
		port:     port,
	}

	// Register HTTP metrics
	registry.MustRegister(m.HTTPRequestsTotal)
	registry.MustRegister(m.HTTPRequestDuration)
	registry.MustRegister(m.MemoryUsage)
	registry.MustRegister(m.Goroutines)

	// Register default Go collectors for CPU, memory, GC stats
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return m
}

// RecordRequest records a gRPC request with its duration and status
func (m *Metrics) RecordRequest(method string, durationMs int64, status string) {
	m.RequestsTotal.WithLabelValues(method, status).Inc()
	m.RequestDuration.WithLabelValues(method, status).Observe(float64(durationMs) / 1000.0)
}

// RecordError records a gRPC error
func (m *Metrics) RecordError(method, errorMsg string) {
	// Truncate error message to prevent high cardinality
	if len(errorMsg) > 100 {
		errorMsg = errorMsg[:100] + "..."
	}
	m.ErrorsTotal.WithLabelValues(method, errorMsg).Inc()
}

// RecordPanic records a gRPC panic
func (m *Metrics) RecordPanic(method, panicMsg string) {
	// Truncate panic message to prevent high cardinality
	if len(panicMsg) > 100 {
		panicMsg = panicMsg[:100] + "..."
	}
	m.PanicsTotal.WithLabelValues(method, panicMsg).Inc()
}

// RecordHTTPRequest records an HTTP request with its duration and status
func (m *Metrics) RecordHTTPRequest(method, path string, durationMs int64, status string) {
	if m.HTTPRequestsTotal != nil {
		m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	}
	if m.HTTPRequestDuration != nil {
		m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(float64(durationMs) / 1000.0)
	}
}

// updateResourceMetrics updates memory and CPU metrics
func (m *Metrics) updateResourceMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.MemoryUsage.WithLabelValues("alloc").Set(float64(memStats.Alloc))
	m.MemoryUsage.WithLabelValues("total_alloc").Set(float64(memStats.TotalAlloc))
	m.MemoryUsage.WithLabelValues("sys").Set(float64(memStats.Sys))
	m.MemoryUsage.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
	m.MemoryUsage.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
	m.MemoryUsage.WithLabelValues("heap_inuse").Set(float64(memStats.HeapInuse))
	m.MemoryUsage.WithLabelValues("stack_inuse").Set(float64(memStats.StackInuse))

	m.Goroutines.Set(float64(runtime.NumGoroutine()))
}

// Start starts the metrics HTTP server
func (m *Metrics) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	m.server = &http.Server{
		Addr:    ":" + m.port,
		Handler: mux,
	}

	log.Printf("starting metrics server on port %s", m.port)

	// Start background goroutine to update resource metrics
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			m.updateResourceMetrics()
			<-ticker.C
		}
	}()

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the metrics server
func (m *Metrics) Stop(ctx context.Context) error {
	if m.server != nil {
		return m.server.Shutdown(ctx)
	}
	return nil
}
