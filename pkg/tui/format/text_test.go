package format

import "testing"

func TestPadRight(t *testing.T) {
	if got := PadRight("a", 3); got != "a  " {
		t.Fatalf("expected padding to width 3, got %q", got)
	}

	if got := PadRight("abcd", 2); got != "ab" {
		t.Fatalf("expected truncation to width 2, got %q", got)
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
