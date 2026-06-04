package observability

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type LogConfig struct {
	Format string
	Output string
	File   string
	Level  string
}

func NewLogger(cfg LogConfig) (zerolog.Logger, io.Closer, error) {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	level := zerolog.InfoLevel
	if cfg.Level != "" {
		parsed, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
		if err != nil {
			return zerolog.Logger{}, nil, fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
		}
		level = parsed
	}

	var writer io.Writer = os.Stdout
	var closer io.Closer
	if strings.EqualFold(cfg.Output, "file") {
		if cfg.File == "" {
			return zerolog.Logger{}, nil, fmt.Errorf("LOG_FILE is required when LOG_OUTPUT=file")
		}
		f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return zerolog.Logger{}, nil, fmt.Errorf("open log file: %w", err)
		}
		writer = f
		closer = f
	}

	if strings.EqualFold(cfg.Format, "console") {
		writer = zerolog.ConsoleWriter{Out: writer, TimeFormat: time.RFC3339}
	}

	logger := zerolog.New(writer).Level(level).With().Timestamp().Logger()
	return logger, closer, nil
}
