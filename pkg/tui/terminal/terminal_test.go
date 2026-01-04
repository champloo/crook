package terminal

import (
	"os"
	"testing"
)

func TestDetectCapabilities_Defaults(t *testing.T) {
	// Save and restore environment
	origTerm := os.Getenv("TERM")
	origColorTerm := os.Getenv("COLORTERM")
	origNoColor := os.Getenv("NO_COLOR")
	origTmux := os.Getenv("TMUX")
	defer func() {
		_ = os.Setenv("TERM", origTerm)
		_ = os.Setenv("COLORTERM", origColorTerm)
		_ = os.Setenv("NO_COLOR", origNoColor)
		_ = os.Setenv("TMUX", origTmux)
	}()

	tests := []struct {
		name         string
		term         string
		colorTerm    string
		noColor      string
		tmux         string
		want256      bool
		want16       bool
		wantNoColors bool
		wantUnicode  bool
		wantTmux     bool
	}{
		{
			name:         "dumb terminal",
			term:         "dumb",
			wantNoColors: true,
			wantUnicode:  false,
		},
		{
			name:         "empty TERM",
			term:         "",
			wantNoColors: true,
			wantUnicode:  false,
		},
		{
			name:        "xterm-256color",
			term:        "xterm-256color",
			want256:     true,
			wantUnicode: true,
		},
		{
			name:        "truecolor",
			term:        "xterm",
			colorTerm:   "truecolor",
			want256:     true,
			wantUnicode: true,
		},
		{
			name:        "24bit",
			term:        "xterm",
			colorTerm:   "24bit",
			want256:     true,
			wantUnicode: true,
		},
		{
			name:        "xterm basic",
			term:        "xterm",
			want16:      true,
			wantUnicode: true,
		},
		{
			name:        "screen",
			term:        "screen",
			want16:      true,
			wantUnicode: true,
		},
		{
			name:        "tmux-256color",
			term:        "tmux-256color",
			want256:     true,
			wantUnicode: true,
		},
		{
			name:        "linux console",
			term:        "linux",
			want16:      true,
			wantUnicode: true,
		},
		{
			name:         "NO_COLOR overrides",
			term:         "xterm-256color",
			noColor:      "1",
			wantNoColors: true,
			wantUnicode:  true, // NO_COLOR only affects colors, not Unicode
		},
		{
			name:        "tmux detected",
			term:        "xterm-256color",
			tmux:        "/tmp/tmux-123/default,12345,0",
			want256:     true,
			wantUnicode: true,
			wantTmux:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TERM", tt.term)
			_ = os.Setenv("COLORTERM", tt.colorTerm)
			if tt.noColor != "" {
				_ = os.Setenv("NO_COLOR", tt.noColor)
			} else {
				_ = os.Unsetenv("NO_COLOR")
			}
			if tt.tmux != "" {
				_ = os.Setenv("TMUX", tt.tmux)
			} else {
				_ = os.Unsetenv("TMUX")
			}

			cap := DetectCapabilities()

			if cap.Has256Colors != tt.want256 {
				t.Errorf("Has256Colors = %v, want %v", cap.Has256Colors, tt.want256)
			}
			if cap.Has16Colors != tt.want16 {
				t.Errorf("Has16Colors = %v, want %v", cap.Has16Colors, tt.want16)
			}
			if cap.HasNoColors != tt.wantNoColors {
				t.Errorf("HasNoColors = %v, want %v", cap.HasNoColors, tt.wantNoColors)
			}
			if cap.HasUnicode != tt.wantUnicode {
				t.Errorf("HasUnicode = %v, want %v", cap.HasUnicode, tt.wantUnicode)
			}
			if cap.IsTmux != tt.wantTmux {
				t.Errorf("IsTmux = %v, want %v", cap.IsTmux, tt.wantTmux)
			}
		})
	}
}

func TestDetectCapabilities_Screen(t *testing.T) {
	origTerm := os.Getenv("TERM")
	defer func() { _ = os.Setenv("TERM", origTerm) }()

	_ = os.Setenv("TERM", "screen.xterm-256color")
	cap := DetectCapabilities()

	if !cap.IsScreen {
		t.Error("IsScreen should be true for screen.xterm-256color")
	}
}

func TestGetIcons_Unicode(t *testing.T) {
	cap := Capability{HasUnicode: true, HasNoColors: false}
	icons := GetIcons(cap)

	if icons.Checkmark != "✓" {
		t.Errorf("Checkmark = %q, want ✓", icons.Checkmark)
	}
	if icons.Cross != "✗" {
		t.Errorf("Cross = %q, want ✗", icons.Cross)
	}
}

func TestGetIcons_ASCII(t *testing.T) {
	cap := Capability{HasNoColors: true}
	icons := GetIcons(cap)

	if icons.Checkmark != "[OK]" {
		t.Errorf("Checkmark = %q, want [OK]", icons.Checkmark)
	}
	if icons.Cross != "[X]" {
		t.Errorf("Cross = %q, want [X]", icons.Cross)
	}
}

func TestGetIcons_NoUnicode(t *testing.T) {
	cap := Capability{HasUnicode: false, HasNoColors: false}
	icons := GetIcons(cap)

	if icons.Checkmark != "[OK]" {
		t.Errorf("Checkmark = %q, want [OK]", icons.Checkmark)
	}
}

func TestSpinnerFrames_Unicode(t *testing.T) {
	cap := Capability{HasUnicode: true, HasNoColors: false}
	frames := SpinnerFrames(cap)

	if len(frames) != 4 {
		t.Errorf("expected 4 spinner frames, got %d", len(frames))
	}
	if frames[0] != "◐" {
		t.Errorf("expected Unicode spinner, got %q", frames[0])
	}
}

func TestSpinnerFrames_ASCII(t *testing.T) {
	cap := Capability{HasNoColors: true}
	frames := SpinnerFrames(cap)

	if len(frames) != 4 {
		t.Errorf("expected 4 spinner frames, got %d", len(frames))
	}
	if frames[0] != "-" {
		t.Errorf("expected ASCII spinner, got %q", frames[0])
	}
}

func TestGetProgressChars_Unicode(t *testing.T) {
	cap := Capability{HasUnicode: true, HasNoColors: false}
	chars := GetProgressChars(cap)

	if chars.Full != "█" {
		t.Errorf("Full = %q, want █", chars.Full)
	}
	if chars.Empty != "░" {
		t.Errorf("Empty = %q, want ░", chars.Empty)
	}
}

func TestGetProgressChars_ASCII(t *testing.T) {
	cap := Capability{HasNoColors: true}
	chars := GetProgressChars(cap)

	if chars.Full != "=" {
		t.Errorf("Full = %q, want =", chars.Full)
	}
	if chars.Empty != "-" {
		t.Errorf("Empty = %q, want -", chars.Empty)
	}
}

func TestIsTooNarrow(t *testing.T) {
	tests := []struct {
		width int
		want  bool
	}{
		{0, false},   // 0 means unknown, don't warn
		{79, true},   // below minimum
		{80, false},  // exactly minimum
		{120, false}, // above minimum
	}

	for _, tt := range tests {
		got := IsTooNarrow(tt.width)
		if got != tt.want {
			t.Errorf("IsTooNarrow(%d) = %v, want %v", tt.width, got, tt.want)
		}
	}
}

func TestIsTooShort(t *testing.T) {
	tests := []struct {
		height int
		want   bool
	}{
		{0, false},  // 0 means unknown, don't warn
		{23, true},  // below minimum
		{24, false}, // exactly minimum
		{40, false}, // above minimum
	}

	for _, tt := range tests {
		got := IsTooShort(tt.height)
		if got != tt.want {
			t.Errorf("IsTooShort(%d) = %v, want %v", tt.height, got, tt.want)
		}
	}
}

func TestSizeWarning(t *testing.T) {
	tests := []struct {
		width   int
		height  int
		wantMsg bool
	}{
		{80, 24, false},  // OK
		{79, 24, true},   // narrow
		{80, 23, true},   // short
		{79, 23, true},   // both
		{0, 0, false},    // unknown
		{120, 40, false}, // large
	}

	for _, tt := range tests {
		msg := SizeWarning(tt.width, tt.height)
		gotMsg := msg != ""
		if gotMsg != tt.wantMsg {
			t.Errorf("SizeWarning(%d, %d) = %q, wantMsg=%v", tt.width, tt.height, msg, tt.wantMsg)
		}
	}
}

func TestSizeWarning_Content(t *testing.T) {
	// Test narrow warning content
	msg := SizeWarning(60, 30)
	if msg == "" {
		t.Fatal("expected warning for narrow terminal")
	}
	if !containsStr(msg, "narrow") {
		t.Errorf("warning should mention 'narrow', got %q", msg)
	}

	// Test short warning content
	msg = SizeWarning(100, 20)
	if msg == "" {
		t.Fatal("expected warning for short terminal")
	}
	if !containsStr(msg, "short") {
		t.Errorf("warning should mention 'short', got %q", msg)
	}

	// Test both warnings
	msg = SizeWarning(60, 20)
	if !containsStr(msg, "narrow") || !containsStr(msg, "short") {
		t.Errorf("warning should mention both 'narrow' and 'short', got %q", msg)
	}
}

func TestConfigureLipgloss(t *testing.T) {
	// Just verify it doesn't panic
	caps := []Capability{
		{HasNoColors: true},
		{Has256Colors: true},
		{Has16Colors: true},
	}

	for _, cap := range caps {
		ConfigureLipgloss(cap)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
