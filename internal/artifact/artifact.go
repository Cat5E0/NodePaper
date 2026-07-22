// Package artifact defines files produced or managed by NodePaper operations.
package artifact

// Artifact identifies a user-visible or diagnostic file associated with an
// operation. Path is absolute unless an operation explicitly documents a
// project-relative path.
type Artifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}
