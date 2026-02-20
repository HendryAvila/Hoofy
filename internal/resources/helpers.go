package resources

import (
	"fmt"
	"os"
	"path/filepath"
)

// findRoot walks up from cwd looking for sdd/sdd.json.
// Shared utility for resource handlers.
func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	current := dir
	for {
		candidate := filepath.Join(current, "sdd", "sdd.json")
		if _, err := os.Stat(candidate); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return dir, nil
		}
		current = parent
	}
}
