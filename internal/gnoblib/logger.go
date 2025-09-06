package gnoblib

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

var (
	Logger = defaultLogger()
)

func SetLogger(logger *slog.Logger) {
	Logger = logger
}

type _logHandler struct {
	start  time.Time
	output io.Writer
	*slog.TextHandler
}

func (h *_logHandler) Handle(_ context.Context, r slog.Record) error {
	var line strings.Builder
	dur := r.Time.Sub(h.start)
	line.WriteString(fmt.Sprintf("[%04d] ", int(dur.Seconds())))
	switch r.Level {
	case slog.LevelDebug:
		line.WriteString("DEBUG ")
	case slog.LevelInfo:
		line.WriteString("INFO  ")
	case slog.LevelWarn:
		line.WriteString("WARN  ")
	case slog.LevelError:
		line.WriteString("ERROR ")
	}
	line.WriteString(r.Message)
	first := true
	if n := r.NumAttrs(); n > 0 {
		line.WriteString(" {")
		r.Attrs(func(attr slog.Attr) bool {
			if !first {
				line.WriteByte(' ')
			}
			line.WriteString(attr.String())
			first = false
			return true
		})
		line.WriteByte('}')
	}
	line.WriteString("\n")
	_, err := h.output.Write([]byte(line.String()))
	return err
}

func defaultLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	switch os.Getenv(EnvLogLevel) {
	case "debug":
		opts.Level = slog.LevelDebug
	case "info":
		opts.Level = slog.LevelInfo
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	}
	return slog.New(
		&_logHandler{
			start:       time.Now(),
			output:      os.Stderr,
			TextHandler: slog.NewTextHandler(os.Stderr, opts),
		},
	)
}
