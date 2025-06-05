package logging

import (
	"log/slog"
	"os"
	"sync"
)

// Make sure that the logger is initialized only once
var once sync.Once

// Config for the logger
type LoggerConfig struct {
	Level     slog.Leveler
	Format    string // Should be "text" or "json"
	Filename  string // Empty for stdout
	AddSource bool   // Whether to include source info in logs
}

// InitLogger creates a slog.Logger instance and sets it as the default logger.
func InitLogger(cfg LoggerConfig) {
	once.Do(func() {
		var output *os.File
		if cfg.Filename != "" {
			var err error
			output, err = os.OpenFile(cfg.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				output = os.Stdout
				os.Stderr.WriteString("Failed to open log file, using stdout: " + err.Error() + "\n")
			}
		} else {
			output = os.Stdout
		}
		opts := &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		}
		var handler slog.Handler
		switch cfg.Format {
		case "json":
			handler = slog.NewJSONHandler(output, opts)
		default:
			handler = slog.NewTextHandler(output, opts)
		}
		slog.SetDefault(slog.New(handler))
	})
}
