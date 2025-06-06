package logging

import (
	"path/filepath"
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
	ShortPath bool   // Print only filename is ource info logs
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
		replace := func(groups []string, a slog.Attr) slog.Attr {
			// Remove the directory from the source's filename.
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				source.File = filepath.Base(source.File)
			}
			return a
		}
		opts := &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		}
		if cfg.AddSource && cfg.ShortPath {
			opts.ReplaceAttr = replace
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
