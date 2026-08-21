//go:build !windows

package export

import "path/filepath"

func canonicalExistingPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
