package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andri/crook/internal/logger"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config logger.Config
	}{
		{
			name: "text format",
			config: logger.Config{
				Level:  logger.LevelInfo,
				Format: logger.FormatText,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "json format",
			config: logger.Config{
				Level:  logger.LevelDebug,
				Format: logger.FormatJSON,
				Output: &bytes.Buffer{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := logger.New(tt.config)
			if l == nil {
				t.Fatal("expected logger to be non-nil")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name         string
		level        logger.Level
		logFunc      func(*logger.Logger, string)
		shouldAppear bool
	}{
		{
			name:         "debug level logs debug",
			level:        logger.LevelDebug,
			logFunc:      func(l *logger.Logger, msg string) { l.Debug(msg) },
			shouldAppear: true,
		},
		{
			name:         "info level filters debug",
			level:        logger.LevelInfo,
			logFunc:      func(l *logger.Logger, msg string) { l.Debug(msg) },
			shouldAppear: false,
		},
		{
			name:         "info level logs info",
			level:        logger.LevelInfo,
			logFunc:      func(l *logger.Logger, msg string) { l.Info(msg) },
			shouldAppear: true,
		},
		{
			name:         "warn level filters info",
			level:        logger.LevelWarn,
			logFunc:      func(l *logger.Logger, msg string) { l.Info(msg) },
			shouldAppear: false,
		},
		{
			name:         "error level logs error",
			level:        logger.LevelError,
			logFunc:      func(l *logger.Logger, msg string) { l.Error(msg) },
			shouldAppear: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := logger.New(logger.Config{
				Level:  tt.level,
				Format: logger.FormatText,
				Output: buf,
			})

			testMsg := "test message"
			tt.logFunc(l, testMsg)

			output := buf.String()
			if tt.shouldAppear && !strings.Contains(output, testMsg) {
				t.Errorf("expected log message to appear but it didn't: %s", output)
			}
			if !tt.shouldAppear && strings.Contains(output, testMsg) {
				t.Errorf("expected log message to be filtered but it appeared: %s", output)
			}
		})
	}
}

func TestJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatJSON,
		Output: buf,
	})

	l.Info("test message", "key", "value")
	output := buf.String()

	// JSON output should contain the message and key-value pair
	if !strings.Contains(output, "test message") {
		t.Errorf("expected JSON to contain message, got: %s", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("expected JSON to contain key-value pair, got: %s", output)
	}
}

func TestTextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: buf,
	})

	l.Info("test message", "key", "value")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("expected text to contain message, got: %s", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("expected text to contain key-value pair, got: %s", output)
	}
}

func TestWith(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: buf,
	})

	contextLogger := l.With("component", "test")
	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component") || !strings.Contains(output, "test") {
		t.Errorf("expected context to be included, got: %s", output)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	original := logger.GetDefault()
	defer func() { logger.SetDefault(original) }()

	logger.SetDefault(logger.New(logger.Config{
		Level:  logger.LevelDebug,
		Format: logger.FormatText,
		Output: buf,
	}))

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	output := buf.String()
	messages := []string{"debug msg", "info msg", "warn msg", "error msg"}
	for _, msg := range messages {
		if !strings.Contains(output, msg) {
			t.Errorf("expected output to contain %q, got: %s", msg, output)
		}
	}
}
