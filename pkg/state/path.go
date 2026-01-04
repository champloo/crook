package state

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ResolvePath resolves the state file path using the template and node name.
func ResolvePath(templatePath, node string) (string, error) {
	if strings.TrimSpace(templatePath) == "" {
		return "", errors.New("state file template is required")
	}
	if strings.TrimSpace(node) == "" {
		return "", errors.New("state node is required")
	}

	tmpl, err := template.New("state-path").Option("missingkey=error").Parse(templatePath)
	if err != nil {
		return "", fmt.Errorf("invalid state file template: %w", err)
	}

	var rendered bytes.Buffer
	if execErr := tmpl.Execute(&rendered, struct{ Node string }{Node: node}); execErr != nil {
		return "", fmt.Errorf("invalid state file template: %w", execErr)
	}

	resolved, err := expandHome(rendered.String())
	if err != nil {
		return "", err
	}

	if ensureErr := ensureParentDir(resolved); ensureErr != nil {
		return "", ensureErr
	}

	return resolved, nil
}

// ResolvePathWithOverride uses an explicit path when provided, otherwise resolves the template.
func ResolvePathWithOverride(explicitPath, templatePath, node string) (string, error) {
	if strings.TrimSpace(explicitPath) != "" {
		resolved, err := expandHome(explicitPath)
		if err != nil {
			return "", err
		}
		if ensureErr := ensureParentDir(resolved); ensureErr != nil {
			return "", ensureErr
		}
		return resolved, nil
	}

	return ResolvePath(templatePath, node)
}

func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create state file directory %s: %w", dir, err)
	}
	return nil
}
