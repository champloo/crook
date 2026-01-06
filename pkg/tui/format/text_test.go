package format

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPadRight(t *testing.T) {
	if got := PadRight("a", 3); got != "a  " {
		t.Fatalf("expected padding to width 3, got %q", got)
	}

	if got := PadRight("abcd", 2); got != "ab" {
		t.Fatalf("expected truncation to width 2, got %q", got)
	}
}

func TestPadRight_ANSIAndWideRunes(t *testing.T) {
	styled := "\x1b[31mab\x1b[0m"
	got := PadRight(styled, 4)
	if w := ansi.StringWidth(got); w != 4 {
		t.Fatalf("expected display width 4, got %d (%q)", w, got)
	}
	if stripped := ansi.Strip(got); stripped != "ab  " {
		t.Fatalf("expected stripped output %q, got %q", "ab  ", stripped)
	}
	if strings.Contains(ansi.Strip(got), "\x1b") {
		t.Fatalf("expected stripped output to contain no ESC bytes")
	}

	wide := "你好" // width 4
	got = PadRight(wide, 5)
	if w := ansi.StringWidth(got); w != 5 {
		t.Fatalf("expected display width 5, got %d (%q)", w, got)
	}
	if stripped := ansi.Strip(got); stripped != "你好 " {
		t.Fatalf("expected stripped output %q, got %q", "你好 ", stripped)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("abc", 0); got != "" {
		t.Fatalf("expected empty string for width 0, got %q", got)
	}

	if got := Truncate("abc", 2); got != "ab" {
		t.Fatalf("expected truncation to width 2, got %q", got)
	}
}

func TestTruncate_ANSI(t *testing.T) {
	styled := "\x1b[31mabcdef\x1b[0m"
	got := Truncate(styled, 3)
	if stripped := ansi.Strip(got); stripped != "abc" {
		t.Fatalf("expected stripped output %q, got %q", "abc", stripped)
	}
	if strings.Contains(ansi.Strip(got), "\x1b") {
		t.Fatalf("expected stripped output to contain no ESC bytes")
	}
}
