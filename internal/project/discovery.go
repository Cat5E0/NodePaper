package project

import (
	"fmt"
	"os"
	"path/filepath"

	"nodepaper/internal/diagnostic"
)

const (
	CodeProjectNotFound       = "NP1001"
	CodeProjectNotDirectory   = "NP1002"
	CodeProjectPathUnreadable = "NP1003"
)

// DiscoveryError preserves both a user-facing diagnostic and the underlying
// filesystem error, when one exists.
type DiscoveryError struct {
	Diagnostic diagnostic.Diagnostic
	Cause      error
}

func (e *DiscoveryError) Error() string {
	return e.Diagnostic.Message
}

func (e *DiscoveryError) Unwrap() error {
	return e.Cause
}

// Discover resolves an explicit project directory or discovers a project by
// walking upward from the process working directory.
func Discover(projectDir string) (Project, error) {
	if filepath.IsAbs(projectDir) {
		return DiscoverFrom(projectDir, "")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return Project{}, discoveryError(
			CodeProjectPathUnreadable,
			"Unable to read the current working directory.",
			"",
			"Run NodePaper from an accessible directory or pass an explicit project directory.",
			err,
		)
	}
	return DiscoverFrom(projectDir, workingDir)
}

// DiscoverFrom is Discover with an explicit working directory. It lets callers
// and tests avoid changing process-global working directory state.
func DiscoverFrom(projectDir, workingDir string) (Project, error) {
	if projectDir != "" {
		root, err := absoluteFrom(projectDir, workingDir)
		if err != nil {
			return Project{}, discoveryError(
				CodeProjectPathUnreadable,
				fmt.Sprintf("Unable to resolve project directory %q.", projectDir),
				projectDir,
				"Pass an existing project directory containing nodepaper.yaml.",
				err,
			)
		}
		return inspectExplicit(root)
	}

	start, err := absoluteFrom(workingDir, "")
	if err != nil {
		return Project{}, discoveryError(
			CodeProjectPathUnreadable,
			fmt.Sprintf("Unable to resolve working directory %q.", workingDir),
			workingDir,
			"Run NodePaper from an accessible directory or pass an explicit project directory.",
			err,
		)
	}

	info, err := os.Stat(start)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, notFound(start, err)
		}
		return Project{}, unreadable(start, err)
	}
	if !info.IsDir() {
		return Project{}, notDirectory(start, nil)
	}

	for current := start; ; current = filepath.Dir(current) {
		configPath := filepath.Join(current, MarkerFile)
		info, statErr := os.Stat(configPath)
		switch {
		case statErr == nil && info.Mode().IsRegular():
			return Project{Root: current, ConfigPath: configPath}, nil
		case statErr == nil:
			return Project{}, discoveryError(
				CodeProjectNotDirectory,
				fmt.Sprintf("Project marker is not a regular file: %s", configPath),
				configPath,
				"Replace it with a nodepaper.yaml file.",
				nil,
			)
		case !os.IsNotExist(statErr):
			return Project{}, unreadable(configPath, statErr)
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}

	return Project{}, notFound(start, nil)
}

func inspectExplicit(root string) (Project, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, notFound(root, err)
		}
		return Project{}, unreadable(root, err)
	}
	if !info.IsDir() {
		return Project{}, notDirectory(root, nil)
	}

	configPath := filepath.Join(root, MarkerFile)
	info, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, notFound(root, err)
		}
		return Project{}, unreadable(configPath, err)
	}
	if !info.Mode().IsRegular() {
		return Project{}, discoveryError(
			CodeProjectNotDirectory,
			fmt.Sprintf("Project marker is not a regular file: %s", configPath),
			configPath,
			"Replace it with a nodepaper.yaml file.",
			nil,
		)
	}

	return Project{Root: root, ConfigPath: configPath}, nil
}

func absoluteFrom(path, base string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		if base == "" {
			return filepath.Abs(path)
		}
		path = filepath.Join(base, path)
	}
	return filepath.Abs(filepath.Clean(path))
}

func notFound(path string, cause error) error {
	return discoveryError(
		CodeProjectNotFound,
		fmt.Sprintf("NodePaper project not found from %s.", path),
		path,
		"Pass a directory containing nodepaper.yaml or run the command inside a NodePaper project.",
		cause,
	)
}

func notDirectory(path string, cause error) error {
	return discoveryError(
		CodeProjectNotDirectory,
		fmt.Sprintf("Project path is not a directory: %s", path),
		path,
		"Pass the Project directory, not a Markdown file or nodepaper.yaml itself.",
		cause,
	)
}

func unreadable(path string, cause error) error {
	return discoveryError(
		CodeProjectPathUnreadable,
		fmt.Sprintf("Unable to inspect project path: %s", path),
		path,
		"Check that the path exists and is accessible.",
		cause,
	)
}

func discoveryError(code, message, file, suggestion string, cause error) error {
	return &DiscoveryError{
		Diagnostic: diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       code,
			Message:    message,
			File:       file,
			Suggestion: suggestion,
			Source:     "project",
		},
		Cause: cause,
	}
}
