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
}

var version = "dev"

func main() {
	root := &cobra.Command{Use: "openapi-mock", Short: "OpenAPI mock server"}

	run := &cobra.Command{
		Use: "run", Short: "Run server", Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg Config
			env.Parse(&cfg)

			if v, _ := cmd.Flags().GetString("port"); v != "" {
				cfg.Port = v
			}
			if len(args) >= 1 {
				cfg.Host = args[0]
			}
			if len(args) >= 2 {
				cfg.Port = args[1]
			}

			return runServer(cfg)
		},
	}
	run.Flags().StringP("port", "p", "", "Port")

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

	var m *metrics.Metrics
	if cfg.EnableMetrics {
		m = metrics.NewHTTP(cfg.MetricsPort)
		m.Start()
	}

	if cfg.EnableMgmt {
		mgmt.New(rec, cfg.MgmtPort).Start()
	}

	// Build middlewares
	middlewares := []func(http.Handler) http.Handler{
		middleware.Recording(rec, m, cfg.EnableLogging),
	}

	// Build app via wire (handles all routing)
	httpApp, err := app.InitializeHTTPApp(middlewares, m, cfg.EnableLogging)
	if err != nil {
		return fmt.Errorf("failed to initialize app: %w", err)
	}

	server := &http.Server{Addr: addr, Handler: httpApp.Router}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
