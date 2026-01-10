package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andri/crook/pkg/cli"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		skipPrompt bool
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "yes lowercase",
			input:      "y\n",
			wantResult: true,
		},
		{
			name:       "yes uppercase",
			input:      "Y\n",
			wantResult: true,
		},
		{
			name:       "yes full word",
			input:      "yes\n",
			wantResult: true,
		},
		{
			name:       "yes full word uppercase",
			input:      "YES\n",
			wantResult: true,
		},
		{
			name:       "no lowercase",
			input:      "n\n",
			wantResult: false,
		},
		{
			name:       "no uppercase",
			input:      "N\n",
			wantResult: false,
		},
		{
			name:       "no full word",
			input:      "no\n",
			wantResult: false,
		},
		{
			name:       "empty input",
			input:      "\n",
			wantResult: false,
		},
		{
			name:       "random input",
			input:      "maybe\n",
			wantResult: false,
		},
		{
			name:       "skip prompt returns true",
			input:      "",
			skipPrompt: true,
			wantResult: true,
		},
		{
			name:       "whitespace around yes",
			input:      "  y  \n",
			wantResult: true,
		},
		{
			name:       "eof without input",
			input:      "",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}

			opts := cli.ConfirmOptions{
				Question:   "Proceed?",
				SkipPrompt: tt.skipPrompt,
				Input:      input,
				Output:     output,
			}

			result, err := cli.Confirm(opts)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.wantResult {
				t.Errorf("got result %v, want %v", result, tt.wantResult)
			}

			// Verify prompt was written (unless skipped)
			if !tt.skipPrompt {
				if !strings.Contains(output.String(), "Proceed?") {
					t.Errorf("expected prompt in output, got: %s", output.String())
				}
				if !strings.Contains(output.String(), "(y/N)") {
					t.Errorf("expected (y/N) in output, got: %s", output.String())
				}
			}
		})
	}
}

func TestConfirm_DefaultsToStdio(t *testing.T) {
	// Just verify that nil input/output doesn't panic
	// We can't easily test stdin/stdout in unit tests
	opts := cli.ConfirmOptions{
		Question:   "Test?",
		SkipPrompt: true,
	}

	result, err := cli.Confirm(opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true when skip is set")
	}
}
