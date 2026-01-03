package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "crook-state-{{.Node}}.json")

	resolved, err := ResolvePath(templatePath, "worker-01")
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}

	expected := filepath.Join(dir, "crook-state-worker-01.json")
	if resolved != expected {
		t.Fatalf("expected %s, got %s", expected, resolved)
	}
}

func TestResolvePathCreatesDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "state-{{.Node}}.json")

	resolved, err := ResolvePath(target, "node-a")
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}

	info, err := os.Stat(filepath.Dir(resolved))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory")
	}
}

func TestResolvePathWithOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.json")

	resolved, err := ResolvePathWithOverride(explicit, "ignored", "node")
	if err != nil {
		t.Fatalf("resolve override: %v", err)
	}
	if resolved != explicit {
		t.Fatalf("expected %s, got %s", explicit, resolved)
	}
}

func TestResolvePathRejectsEmpty(t *testing.T) {
	t.Parallel()

	if _, err := ResolvePath(" ", "node"); err == nil {
		t.Fatalf("expected error for empty template")
	}
	if _, err := ResolvePath("state.json", " "); err == nil {
		t.Fatalf("expected error for empty node")
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	originalHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home: %v", err)
	}
	if originalHome == home {
		t.Skip("temp home matches user home, cannot validate expansion")
	}

	resolved, err := ResolvePath("~/state-{{.Node}}.json", "node")
	if err != nil {
		t.Fatalf("resolve home: %v", err)
	}

	if resolved != filepath.Join(home, "state-node.json") {
		t.Fatalf("expected resolved path %s, got %s", filepath.Join(home, "state-node.json"), resolved)
	}
}
