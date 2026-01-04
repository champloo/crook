package logger

import (
	"io"
	"log/slog"
	"os"
	"sync/atomic"
)

// Logger wraps slog.Logger with a simplified API
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level  Level
	Format Format
	Output io.Writer
}

// Level represents log level
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Format represents output format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Default logger instance
var defaultLogger atomic.Value // stores *Logger

func init() {
	defaultLogger.Store(New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: os.Stderr,
	}))
}

// New creates a new logger with the given configuration
func New(cfg Config) *Logger {
	var handler slog.Handler

	// Default to stderr if no output specified
	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	// Convert custom level to slog.Level
	level := levelToSlog(cfg.Level)

	// Choose handler based on format
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// levelToSlog converts custom Level to slog.Level
func levelToSlog(l Level) slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// With returns a logger with the given attributes
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// Package-level convenience functions using default logger

// Debug logs a debug message
func Debug(msg string, args ...any) {
	l, _ := defaultLogger.Load().(*Logger)
	l.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	l, _ := defaultLogger.Load().(*Logger)
	l.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	l, _ := defaultLogger.Load().(*Logger)
	l.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	l, _ := defaultLogger.Load().(*Logger)
	l.Error(msg, args...)
}

// With returns a logger with the given attributes
func With(args ...any) *Logger {
	l, _ := defaultLogger.Load().(*Logger)
	return l.With(args...)
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger.Store(l)
}

// GetDefault returns the default logger
func GetDefault() *Logger {
	l, _ := defaultLogger.Load().(*Logger)
	return l
}
