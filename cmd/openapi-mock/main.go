package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/spf13/cobra"

	"openapi-mock/internal/app"
	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/mgmt"
	"openapi-mock/pkg/middleware"
	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/observability"
	"openapi-mock/pkg/recorder"
)

type Config struct {
	Host          string `env:"HOST" envDefault:"0.0.0.0"`
	Port          string `env:"PORT" envDefault:"8080"`
	MgmtPort      string `env:"MGMT_PORT" envDefault:"9000"`
	MetricsPort   string `env:"METRICS_PORT" envDefault:"9100"`
	EnableMgmt    bool   `env:"MGMT_ENABLED" envDefault:"true"`
	EnableMetrics bool   `env:"METRICS_ENABLED" envDefault:"true"`
	EnableLogging bool   `env:"HTTP_LOGGING" envDefault:"true"`

	RequestIDHeaders        string `env:"REQUEST_ID_HEADERS" envDefault:"X-Request-ID,X-Request-Id,X-Correlation-ID"`
	RequestIDResponseHeader string `env:"REQUEST_ID_RESPONSE_HEADER" envDefault:"X-Request-ID"`
	CORSAllowOrigins        string `env:"CORS_ALLOW_ORIGINS" envDefault:"*"`

	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`
	LogOutput string `env:"LOG_OUTPUT" envDefault:"stdout"`
	LogFile   string `env:"LOG_FILE" envDefault:""`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`

	TraceEnabled       bool    `env:"TRACE_ENABLED" envDefault:"false"`
	TraceExporter      string  `env:"TRACE_EXPORTER" envDefault:"none"`
	TraceEndpoint      string  `env:"TRACE_ENDPOINT" envDefault:""`
	TraceFile          string  `env:"TRACE_FILE" envDefault:"./traces.json"`
	TraceSamplingRatio float64 `env:"TRACE_SAMPLING_RATIO" envDefault:"1.0"`
}

var version = "dev"

func main() {
	root := &cobra.Command{Use: "openapi-mock", Short: "OpenAPI mock server"}

	run := &cobra.Command{
		Use:   "run",
		Short: "Run server",
		// Backward compat: allow positional HOST PORT.
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg Config
			if err := env.Parse(&cfg); err != nil {
				return fmt.Errorf("failed to parse env: %w", err)
			}

			// Flags > env
			if cmd.Flags().Changed("host") {
				v, _ := cmd.Flags().GetString("host")
				cfg.Host = v
			}
			if cmd.Flags().Changed("port") {
				v, _ := cmd.Flags().GetString("port")
				cfg.Port = v
			}
			if cmd.Flags().Changed("mgmt-port") {
				v, _ := cmd.Flags().GetString("mgmt-port")
				cfg.MgmtPort = v
			}
			if cmd.Flags().Changed("metrics-port") {
				v, _ := cmd.Flags().GetString("metrics-port")
				cfg.MetricsPort = v
			}
			if cmd.Flags().Changed("mgmt-enabled") {
				v, _ := cmd.Flags().GetBool("mgmt-enabled")
				cfg.EnableMgmt = v
			}
			if cmd.Flags().Changed("metrics-enabled") {
				v, _ := cmd.Flags().GetBool("metrics-enabled")
				cfg.EnableMetrics = v
			}
			if cmd.Flags().Changed("http-logging") {
				v, _ := cmd.Flags().GetBool("http-logging")
				cfg.EnableLogging = v
			}
			if cmd.Flags().Changed("logging") {
				v, _ := cmd.Flags().GetBool("logging")
				cfg.EnableLogging = v
			}

			// Logging flag overrides
			if v, _ := cmd.Flags().GetString("log-format"); v != "" {
				cfg.LogFormat = v
			}
			if v, _ := cmd.Flags().GetString("log-output"); v != "" {
				cfg.LogOutput = v
			}
			if v, _ := cmd.Flags().GetString("log-file"); v != "" {
				cfg.LogFile = v
			}
			if v, _ := cmd.Flags().GetString("log-level"); v != "" {
				cfg.LogLevel = v
			}

			// Tracing flag overrides
			if cmd.Flags().Changed("trace-enabled") {
				v, _ := cmd.Flags().GetBool("trace-enabled")
				cfg.TraceEnabled = v
			}
			if v, _ := cmd.Flags().GetString("trace-exporter"); v != "" {
				cfg.TraceExporter = v
			}
			if v, _ := cmd.Flags().GetString("trace-endpoint"); v != "" {
				cfg.TraceEndpoint = v
			}
			if v, _ := cmd.Flags().GetString("trace-file"); v != "" {
				cfg.TraceFile = v
			}
			if cmd.Flags().Changed("trace-sampling-ratio") {
				v, _ := cmd.Flags().GetFloat64("trace-sampling-ratio")
				cfg.TraceSamplingRatio = v
			}

			// Request-ID flag overrides
			if v, _ := cmd.Flags().GetString("request-id-headers"); v != "" {
				cfg.RequestIDHeaders = v
			}
			if v, _ := cmd.Flags().GetString("request-id-response-header"); v != "" {
				cfg.RequestIDResponseHeader = v
			}

			// CORS flag override
			if v, _ := cmd.Flags().GetString("cors-allow-origins"); v != "" {
				cfg.CORSAllowOrigins = v
			}

			// Positional args override too (highest priority), for backwards compatibility.
			if len(args) >= 1 {
				cfg.Host = args[0]
			}
			if len(args) >= 2 {
				cfg.Port = args[1]
			}

			return runServer(cfg)
		},
	}

	// Flags mirror env vars.
	// Note: we intentionally don't set these from env in flag defaults,
	// because env.Parse already did; this keeps precedence simple.
	run.Flags().String("host", "", "Host/interface to bind (env: HOST)")
	run.Flags().StringP("port", "p", "", "HTTP port to bind (env: PORT)")
	run.Flags().String("mgmt-port", "", "Management API port (env: MGMT_PORT)")
	run.Flags().String("metrics-port", "", "Prometheus metrics port (env: METRICS_PORT)")
	run.Flags().Bool("mgmt-enabled", true, "Enable management API (env: MGMT_ENABLED)")
	run.Flags().Bool("metrics-enabled", true, "Enable metrics endpoint (env: METRICS_ENABLED)")
	run.Flags().Bool("http-logging", true, "Enable HTTP request logging (env: HTTP_LOGGING)")
	run.Flags().Bool("logging", true, "Enable request logging (alias for --http-logging, env: HTTP_LOGGING)")

	// Logging flags (parity with grpc-mock)
	run.Flags().String("log-format", "", "Log format: json or console (env: LOG_FORMAT)")
	run.Flags().String("log-output", "", "Log output: stdout or file (env: LOG_OUTPUT)")
	run.Flags().String("log-file", "", "Log file path when output=file (env: LOG_FILE)")
	run.Flags().String("log-level", "", "Log level: debug, info, warn, error (env: LOG_LEVEL)")

	// Tracing flags (parity with grpc-mock)
	run.Flags().Bool("trace-enabled", false, "Enable OpenTelemetry tracing (env: TRACE_ENABLED)")
	run.Flags().String("trace-exporter", "", "Trace exporter: none, file, otlp-http (env: TRACE_EXPORTER)")
	run.Flags().String("trace-endpoint", "", "OTLP HTTP endpoint, e.g. otel-collector:4318 (env: TRACE_ENDPOINT)")
	run.Flags().String("trace-file", "", "Trace file path when exporter=file (env: TRACE_FILE)")
	run.Flags().Float64("trace-sampling-ratio", 0, "Trace sampling ratio 0.0–1.0 (env: TRACE_SAMPLING_RATIO)")

	// Request-ID flags (parity with grpc-mock)
	run.Flags().String("request-id-headers", "", "Comma-separated inbound request-id header names (env: REQUEST_ID_HEADERS)")
	run.Flags().String("request-id-response-header", "", "Response header name for request-id echo (env: REQUEST_ID_RESPONSE_HEADER)")

	// HTTP-specific flag
	run.Flags().String("cors-allow-origins", "", "Comma-separated CORS allow origins (env: CORS_ALLOW_ORIGINS)")

	root.AddCommand(run)
	root.AddCommand(&cobra.Command{
		Use: "version", Run: func(c *cobra.Command, a []string) { fmt.Println(version) },
	})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cfg Config) error {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("Starting HTTP server on %s", addr)

	rec := recorder.New()
	contextValues := mm.NewStore()

	var m *metrics.Metrics
	var mgmtServer *mgmt.Server
	if cfg.EnableMetrics {
		m = metrics.NewHTTP(cfg.MetricsPort)
		_ = m.Start()
	}

	baseLogger, logCloser, err := observability.NewLogger(observability.LogConfig{
		Format: cfg.LogFormat,
		Output: cfg.LogOutput,
		File:   cfg.LogFile,
		Level:  cfg.LogLevel,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	traceShutdown, err := observability.SetupTracing(context.Background(), observability.TraceConfig{
		Enabled:       cfg.TraceEnabled,
		Exporter:      cfg.TraceExporter,
		Endpoint:      cfg.TraceEndpoint,
		File:          cfg.TraceFile,
		SamplingRatio: cfg.TraceSamplingRatio,
		ServiceName:   "openapi-mock",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = traceShutdown(shutdownCtx)
	}()

	// Build middlewares
	middlewares := []func(http.Handler) http.Handler{
		middleware.CORS(middleware.CORSOptions{AllowOrigins: cfg.CORSAllowOrigins}),
		middleware.Recording(rec, m, middleware.RecordingOptions{
			EnableLogging:           cfg.EnableLogging,
			RequestIDHeaders:        observability.NormalizeHeaderList(cfg.RequestIDHeaders, []string{"X-Request-ID", "X-Request-Id", "X-Correlation-ID"}),
			RequestIDResponseHeader: cfg.RequestIDResponseHeader,
			BaseLogger:              baseLogger,
		}),
		middleware.ContextValues(contextValues),
	}

	rt := newMockRuntime(addr, 5*time.Second, func() (http.Handler, error) {
		httpApp, err := app.InitializeHTTPApp(middlewares, m, cfg.EnableLogging)
		if err != nil {
			return nil, err
		}
		return httpApp.Router, nil
	})
	if err := rt.Start(); err != nil {
		return err
	}

	if cfg.EnableMgmt {
		mockHost := cfg.Host
		if mockHost == "" || mockHost == "0.0.0.0" || mockHost == "::" {
			mockHost = "localhost"
		}
		mockServerURL := "http://" + net.JoinHostPort(mockHost, cfg.Port)

		mgmtServer = mgmt.New(mgmt.Options{
			Recorder:      rec,
			ContextValues: contextValues,
			MockDocs:      app.MockDocs(),
			MockServerURL: mockServerURL,
			Port:          cfg.MgmtPort,
			Reset:         resetCallback(rt.Reset, rec, contextValues),
		})
		_ = mgmtServer.Start()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if mgmtServer != nil {
		_ = mgmtServer.Stop(ctx)
	}
	if m != nil {
		_ = m.Stop(ctx)
	}
	return rt.Stop(ctx)
}

func resetCallback(reset func(context.Context) error, rec *recorder.Recorder, values *mm.Store) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := reset(ctx); err != nil {
			return err
		}
		rec.Clear()
		values.Clear()
		return nil
	}
}
