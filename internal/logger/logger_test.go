package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "text format",
			config: Config{
				Level:  LevelInfo,
				Format: FormatText,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "json format",
			config: Config{
				Level:  LevelDebug,
				Format: FormatJSON,
				Output: &bytes.Buffer{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			if logger == nil {
				t.Fatal("expected logger to be non-nil")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name          string
		level         Level
		logFunc       func(*Logger, string)
		shouldAppear  bool
	}{
		{
			name:         "debug level logs debug",
			level:        LevelDebug,
			logFunc:      func(l *Logger, msg string) { l.Debug(msg) },
			shouldAppear: true,
		},
		{
			name:         "info level filters debug",
			level:        LevelInfo,
			logFunc:      func(l *Logger, msg string) { l.Debug(msg) },
			shouldAppear: false,
		},
		{
			name:         "info level logs info",
			level:        LevelInfo,
			logFunc:      func(l *Logger, msg string) { l.Info(msg) },
			shouldAppear: true,
		},
		{
			name:         "warn level filters info",
			level:        LevelWarn,
			logFunc:      func(l *Logger, msg string) { l.Info(msg) },
			shouldAppear: false,
		},
		{
			name:         "error level logs error",
			level:        LevelError,
			logFunc:      func(l *Logger, msg string) { l.Error(msg) },
			shouldAppear: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(Config{
				Level:  tt.level,
				Format: FormatText,
				Output: buf,
			})

			testMsg := "test message"
			tt.logFunc(logger, testMsg)

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
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	})

	logger.Info("test message", "key", "value")
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
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: buf,
	})

	logger.Info("test message", "key", "value")
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
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: buf,
	})

	contextLogger := &Logger{Logger: logger.With("component", "test")}
	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component") || !strings.Contains(output, "test") {
		t.Errorf("expected context to be included, got: %s", output)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	original := defaultLogger
	defer func() { defaultLogger = original }()

	SetDefault(New(Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: buf,
	}))

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	output := buf.String()
	messages := []string{"debug msg", "info msg", "warn msg", "error msg"}
	for _, msg := range messages {
		if !strings.Contains(output, msg) {
			t.Errorf("expected output to contain %q, got: %s", msg, output)
		}
	}
}
