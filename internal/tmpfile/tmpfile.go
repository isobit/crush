// Package tmpfile manages temporary files inside the .crush data directory.
package tmpfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Create creates a temporary file in <dataDir>/tmp/ using the given name
// pattern (see [os.CreateTemp] for pattern semantics). The tmp
// subdirectory is created automatically if it does not exist.
//
// The caller is responsible for closing the returned file.
func Create(dataDir, pattern string) (*os.File, error) {
	dir := filepath.Join(dataDir, "tmp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create tmp dir: %w", err)
	}
	return os.CreateTemp(dir, pattern)
}
