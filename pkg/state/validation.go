package state

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// ValidationWarning represents a non-fatal validation issue.
type ValidationWarning struct {
	Message              string
	RequiresConfirmation bool
}

// ValidationOptions controls state validation behavior.
type ValidationOptions struct {
	MaxAge         time.Duration
	Now            func() time.Time
	SupportedKinds map[string]struct{}
}

// ValidateFile validates a state file on disk and returns parsed state and warnings.
func ValidateFile(path string, opts ValidationOptions) (*State, []ValidationWarning, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil, errors.New("state file path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("state file not found: %s, cannot proceed with up phase", path)
		}
		return nil, nil, fmt.Errorf("stat state file %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, nil, fmt.Errorf("state file path is a directory: %s", path)
	}

	parsed, err := ParseFile(path)
	if err != nil {
		return nil, nil, err
	}

	warnings, err := ValidateState(parsed, path, opts)
	if err != nil {
		return nil, nil, err
	}

	return parsed, warnings, nil
}

// ValidateState validates parsed state data and returns warnings.
func ValidateState(state *State, path string, opts ValidationOptions) ([]ValidationWarning, error) {
	if state == nil {
		return nil, errors.New("state is required")
	}

	supportedKinds := opts.SupportedKinds
	if supportedKinds == nil {
		supportedKinds = map[string]struct{}{
			"Deployment": {},
		}
	}

	for i, resource := range state.Resources {
		fieldPrefix := fmt.Sprintf("resources[%d]", i)
		if strings.TrimSpace(resource.Kind) == "" {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".kind", Message: "missing"}
		}
		if _, ok := supportedKinds[resource.Kind]; !ok {
			return nil, &ValidationError{
				Path:    path,
				Field:   fieldPrefix + ".kind",
				Message: fmt.Sprintf("unsupported kind %q", resource.Kind),
			}
		}
		if strings.TrimSpace(resource.Namespace) == "" {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".namespace", Message: "missing"}
		}
		if strings.TrimSpace(resource.Name) == "" {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".name", Message: "missing"}
		}
		if resource.Replicas < 0 {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".replicas", Message: "must be >= 0"}
		}
	}

	maxAge := opts.MaxAge
	if maxAge == 0 {
		maxAge = 24 * time.Hour
	}

	if !state.Timestamp.IsZero() {
		now := opts.Now
		if now == nil {
			now = time.Now
		}
		age := now().Sub(state.Timestamp)
		if age > maxAge {
			hours := int(math.Round(age.Hours()))
			warnings := []ValidationWarning{
				{
					Message:              fmt.Sprintf("State file is %d hours old. Cluster state may have changed.", hours),
					RequiresConfirmation: true,
				},
			}
			return warnings, nil
		}
	}

	return nil, nil
}

// ResourceExistsFunc checks whether a resource exists.
type ResourceExistsFunc func(ctx context.Context, resource Resource) (bool, error)

// FindMissingResources returns resources that are missing according to the existence check.
func FindMissingResources(ctx context.Context, state *State, exists ResourceExistsFunc) ([]Resource, error) {
	if state == nil {
		return nil, errors.New("state is required")
	}
	if exists == nil {
		return nil, errors.New("resource existence check is required")
	}

	missing := make([]Resource, 0)
	for _, resource := range state.Resources {
		ok, err := exists(ctx, resource)
		if err != nil {
			return nil, err
		}
		if !ok {
			missing = append(missing, resource)
		}
	}

	return missing, nil
}
