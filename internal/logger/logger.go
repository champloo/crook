package logger

import (
	"io"
	"log/slog"
	"os"
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
var defaultLogger *Logger

func init() {
	defaultLogger = New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: os.Stdout,
	})
}

// New creates a new logger with the given configuration
func New(cfg Config) *Logger {
	var handler slog.Handler

	// Convert custom level to slog.Level
	level := levelToSlog(cfg.Level)

	// Choose handler based on format
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
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

// Package-level convenience functions using default logger

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// With returns a logger with the given attributes
func With(args ...any) *Logger {
	return &Logger{
		Logger: defaultLogger.With(args...),
	}
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger = l
}

// GetDefault returns the default logger
func GetDefault() *Logger {
	return defaultLogger
}
