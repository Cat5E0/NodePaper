// Package buildctx creates and manages per-build context including working
// directories, log paths, and build IDs.
package buildctx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// Context holds the file-system scope for a single build.
var buildSequence atomic.Uint64

type Context struct {
	BuildID     string
	ProjectRoot string
	WorkDir     string
	OutputDir   string
	LogDir      string
	LogPath     string
	Sources     []string
	Profile     string
	StartedAt   time.Time
}

// New creates a new build context and ensures required directories exist under
// .nodepaper/. It does not acquire the project lock; the caller must do that.
func New(projectRoot string, sources []string, profile string) (*Context, error) {
	now := time.Now()
	stamp := now.UTC().Format("20060102-150405.000000000")
	sequence := buildSequence.Add(1)
	buildID := fmt.Sprintf("build-%s-p%d-n%d", stamp, os.Getpid(), sequence)

	dotDir := filepath.Join(projectRoot, ".nodepaper")
	workDir := filepath.Join(dotDir, "build")
	logDir := filepath.Join(dotDir, "logs")

	for _, d := range []string{workDir, logDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("create %s: %w", d, err)
		}
	}

	return &Context{
		BuildID:     buildID,
		ProjectRoot: projectRoot,
		WorkDir:     workDir,
		OutputDir:   filepath.Join(projectRoot, "dist"),
		LogDir:      logDir,
		LogPath:     filepath.Join(logDir, buildID+".log"),
		Sources:     sources,
		Profile:     profile,
		StartedAt:   now,
	}, nil
}

// ResolveInWork returns the absolute path under the build working directory
// for a given relative name (e.g. "paper.tex").
func (c *Context) ResolveInWork(name string) string {
	return filepath.Join(c.WorkDir, name)
}

// ResolveOutput returns the absolute output path under dist/.
func (c *Context) ResolveOutput(name string) string {
	return filepath.Join(c.OutputDir, name)
}
