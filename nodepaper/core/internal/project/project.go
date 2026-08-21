// Package project locates and represents NodePaper projects.
package project

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const MarkerFile = "nodepaper.yaml"

var ErrPathOutsideProject = errors.New("path is outside the project root")

// Project is a NodePaper project rooted at the directory containing MarkerFile.
type Project struct {
	Root       string
	ConfigPath string
}

// Resolve resolves path relative to the project root and rejects lexical paths
// outside it. Filesystem consumers must additionally evaluate symlinks when the
// target is required to remain inside the project.
func (p Project) Resolve(path string) (string, error) {
	root, err := filepath.Abs(p.Root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	target := path
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", fmt.Errorf("resolve project path: %w", err)
	}

	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideProject, target)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideProject, target)
	}
	return target, nil
}
