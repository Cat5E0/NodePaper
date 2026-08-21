// Package diagnostic defines structured messages shared by application adapters.
package diagnostic

// Severity describes how a diagnostic affects an operation.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Diagnostic is the stable, structured form of a user-visible problem.
type Diagnostic struct {
	Severity   Severity `json:"severity"`
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
	Source     string   `json:"source,omitempty"`
}
