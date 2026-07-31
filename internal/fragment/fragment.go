// Package fragment validates and snapshots explicitly declared project-local
// LaTeX fragments.
package fragment

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	CodeInvalidDeclaration = "NP2501"
	CodeDuplicate          = "NP2502"
	CodePathEscape         = "NP2503"
	CodeMissing            = "NP2504"
	CodeSymlinkEscape      = "NP2505"
	CodeDocumentCommand    = "NP2506"
	CodeNestedDependency   = "NP2507"
	CodeCommandExecution   = "NP2508"
	CodeUndeclaredInput    = "NP2509"
	CodeChanged            = "NP2510"
)

// File is a validated immutable fragment snapshot.
type File struct {
	Relative string `json:"relative"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
}

// Issue describes one deterministic validation failure.
type Issue struct {
	Code    string
	Path    string
	Line    int
	Message string
}

var commandPatterns = []struct {
	code    string
	message string
	re      *regexp.Regexp
}{
	{CodeDocumentCommand, "document or package command is not allowed", regexp.MustCompile(`(?i)\\(?:documentclass|usepackage|requirepackage|begin\s*\{\s*document\s*\}|end\s*\{\s*document\s*\})`)},
	{CodeNestedDependency, "nested fragment dependency or file read is not allowed", regexp.MustCompile(`(?i)\\(?:input|include|includeonly|inputiffileexists|subfile|import|subimport|includegraphics|verbatiminput|lstinputlisting|bibliography|addbibresource)\b`)},
	{CodeCommandExecution, "TeX I/O, command execution, or command obfuscation is not allowed", regexp.MustCompile(`(?i)\\(?:write18|shellescape|pdfshellescape|immediate|openin|openout|read|write|catcode|csname|scantokens|special|directlua|endlinechar|escapechar)\b`)},
}

// Inspect resolves, validates, reads and hashes declared fragments. It never
// writes project files.
func Inspect(projectRoot string, declarations []string) ([]File, []Issue) {
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, []Issue{{Code: CodeInvalidDeclaration, Message: fmt.Sprintf("cannot resolve Project Root: %v", err)}}
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, []Issue{{Code: CodeInvalidDeclaration, Message: fmt.Sprintf("cannot resolve Project Root links: %v", err)}}
	}

	seen := make(map[string]bool)
	files := make([]File, 0, len(declarations))
	var issues []Issue
	for _, declaration := range declarations {
		rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(declaration)))
		key := strings.ToLower(rel)
		if strings.TrimSpace(declaration) == "" || rel == "." || filepath.IsAbs(rel) || filepath.Ext(rel) != ".tex" {
			issues = append(issues, Issue{Code: CodeInvalidDeclaration, Path: declaration, Message: "fragment must be a non-empty relative .tex path"})
			continue
		}
		if seen[key] {
			issues = append(issues, Issue{Code: CodeDuplicate, Path: declaration, Message: "fragment is declared more than once"})
			continue
		}
		seen[key] = true

		path := filepath.Join(root, rel)
		if !inside(root, path) {
			issues = append(issues, Issue{Code: CodePathEscape, Path: declaration, Message: "fragment path escapes Project Root"})
			continue
		}
		info, err := os.Lstat(path)
		if err != nil {
			issues = append(issues, Issue{Code: CodeMissing, Path: declaration, Message: fmt.Sprintf("fragment is missing: %v", err)})
			continue
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			issues = append(issues, Issue{Code: CodeMissing, Path: declaration, Message: fmt.Sprintf("cannot resolve fragment links: %v", err)})
			continue
		}
		if !inside(realRoot, realPath) {
			issues = append(issues, Issue{Code: CodeSymlinkEscape, Path: declaration, Message: "fragment resolves outside Project Root"})
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			info, err = os.Stat(path)
		}
		if err != nil || !info.Mode().IsRegular() {
			issues = append(issues, Issue{Code: CodeMissing, Path: declaration, Message: "fragment is not a regular file"})
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, Issue{Code: CodeMissing, Path: declaration, Message: fmt.Sprintf("cannot read fragment: %v", err)})
			continue
		}
		if !utf8.Valid(data) {
			issues = append(issues, Issue{Code: CodeInvalidDeclaration, Path: declaration, Message: "fragment must be valid UTF-8"})
			continue
		}
		if issue := inspectCommands(declaration, string(data)); issue != nil {
			issues = append(issues, *issue)
			continue
		}
		sum := sha256.Sum256(data)
		files = append(files, File{Relative: filepath.ToSlash(rel), Path: path, SHA256: hex.EncodeToString(sum[:])})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Relative < files[j].Relative })
	return files, issues
}

// Verify confirms that validated fragments still contain the snapshotted bytes.
func Verify(files []File) []Issue {
	var issues []Issue
	for _, file := range files {
		data, err := os.ReadFile(file.Path)
		if err != nil {
			issues = append(issues, Issue{Code: CodeChanged, Path: file.Relative, Message: fmt.Sprintf("fragment changed or disappeared during build: %v", err)})
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			issues = append(issues, Issue{Code: CodeChanged, Path: file.Relative, Message: "fragment content changed during build"})
		}
	}
	return issues
}

func inspectCommands(path, content string) *Issue {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 4096), len(content)+1)
	line := 0
	var commentCollapsed strings.Builder
	for scanner.Scan() {
		line++
		raw := scanner.Text()
		text := stripTeXComment(raw)
		commentCollapsed.WriteString(text)
		if len(text) == len(raw) {
			commentCollapsed.WriteByte('\n')
		}
		for _, pattern := range commandPatterns {
			if pattern.re.MatchString(text) {
				return &Issue{Code: pattern.code, Path: path, Line: line, Message: pattern.message}
			}
		}
	}
	// A TeX comment removes the physical newline. Check the collapsed stream so
	// commands cannot be hidden as "\\in% comment\nput".
	collapsed := commentCollapsed.String()
	for _, pattern := range commandPatterns {
		if pattern.re.MatchString(collapsed) {
			return &Issue{Code: pattern.code, Path: path, Line: 1, Message: pattern.message}
		}
	}
	return nil
}

func stripTeXComment(line string) string {
	for index := 0; index < len(line); index++ {
		if line[index] != '%' {
			continue
		}
		backslashes := 0
		for previous := index - 1; previous >= 0 && line[previous] == '\\'; previous-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return line[:index]
		}
	}
	return line
}

func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
