// Package validate inspects a NodePaper project for structural and resource
// issues before a build is attempted.
package validate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"nodepaper/internal/config"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/fragment"
	"nodepaper/internal/project"
)

// Result holds the outcome of validating a project.
type Result struct {
	Success     bool
	ProjectRoot string
	Diagnostics []diagnostic.Diagnostic
}

// Run validates the project at projectDir. All paths are resolved against
// the project root.
func Run(ctx context.Context, projectDir string) Result {
	p, err := project.Discover(projectDir)
	if err != nil {
		var diag diagnostic.Diagnostic
		if de, ok := err.(*project.DiscoveryError); ok {
			diag = de.Diagnostic
		} else {
			diag = diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "NP1001",
				Message:  fmt.Sprintf("cannot discover project: %v", err),
				Source:   "validate",
			}
		}
		return Result{Diagnostics: []diagnostic.Diagnostic{diag}}
	}

	result := Result{ProjectRoot: p.Root}
	if appendCancellation(ctx, &result) {
		return result
	}

	cfg, err := config.Load(p.ConfigPath)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, configDiag(err))
		return result
	}

	result.Diagnostics = append(result.Diagnostics, validateConfigPaths(p, cfg)...)
	result.Diagnostics = append(result.Diagnostics, validateSources(p, cfg)...)
	result.Diagnostics = append(result.Diagnostics, validateFragments(p, cfg)...)

	files := readableSources(p, cfg)
	if len(files) > 0 {
		result.Diagnostics = append(result.Diagnostics, validateFrontMatter(files[0].abs, files[0].rel)...)
		result.Diagnostics = append(result.Diagnostics, validateAbstract(files[0].abs, files[0].rel)...)
		result.Diagnostics = append(result.Diagnostics, validateLaterFrontMatter(files[1:])...)
	}

	if !appendCancellation(ctx, &result) {
		result.Diagnostics = append(result.Diagnostics, validateResources(p, files)...)
		result.Diagnostics = append(result.Diagnostics, validateCitationsAndCrossrefs(p, files)...)
		result.Diagnostics = append(result.Diagnostics, validateDeclaredFragmentInputs(files, cfg)...)
		result.Diagnostics = append(result.Diagnostics, validateRawLatex(files)...)
	}

	result.Success = !hasErrors(result.Diagnostics)
	return result
}

func appendCancellation(ctx context.Context, result *Result) bool {
	if err := ctx.Err(); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP9001",
			Message:  fmt.Sprintf("validation cancelled: %v", err),
			Source:   "validate",
		})
		return true
	}
	return false
}

func hasErrors(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

// ---------- configuration and source validation -------------------------

func validateConfigPaths(p project.Project, cfg config.ProjectConfig) []diagnostic.Diagnostic {
	if cfg.Output.File == "" {
		return nil
	}
	if _, err := p.Resolve(cfg.Output.File); err != nil {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP1503",
			Message:    fmt.Sprintf("output path outside project: %s", cfg.Output.File),
			File:       "nodepaper.yaml",
			Suggestion: "Choose an output path inside the project directory.",
			Source:     "validate",
		}}
	}
	return nil
}

func validateFragments(p project.Project, cfg config.ProjectConfig) []diagnostic.Diagnostic {
	_, issues := fragment.Inspect(p.Root, cfg.LatexFragments)
	diags := make([]diagnostic.Diagnostic, 0, len(issues))
	for _, issue := range issues {
		diags = append(diags, fragmentDiagnostic(issue, "validate"))
	}
	return diags
}

func fragmentDiagnostic(issue fragment.Issue, source string) diagnostic.Diagnostic {
	suggestion := "Declare a readable UTF-8 .tex file inside the Project Root and remove unsafe TeX commands."
	if issue.Code == fragment.CodeChanged {
		suggestion = "Do not edit fragments while a build is running; rebuild after the file is stable."
	}
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityError,
		Code:       issue.Code,
		Message:    issue.Message,
		File:       issue.Path,
		Line:       issue.Line,
		Suggestion: suggestion,
		Source:     source,
	}
}

func validateSources(p project.Project, cfg config.ProjectConfig) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	seen := map[string]bool{}

	for _, rel := range cfg.SourceFiles() {
		clean := strings.ToLower(filepath.Clean(rel))
		if seen[clean] {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP2001",
				Message:    fmt.Sprintf("duplicate source: %s", rel),
				File:       rel,
				Suggestion: "Remove the duplicate entry.",
				Source:     "validate",
			})
			continue
		}
		seen[clean] = true

		absPath, err := p.Resolve(rel)
		if err != nil {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP2002",
				Message:    fmt.Sprintf("source path outside project: %s", rel),
				File:       rel,
				Suggestion: "Move the file inside the project directory.",
				Source:     "validate",
			})
			continue
		}

		info, err := os.Stat(absPath)
		if err != nil {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP2003",
				Message:    fmt.Sprintf("source not found: %s", rel),
				File:       rel,
				Suggestion: "Check that the file exists.",
				Source:     "validate",
			})
			continue
		}
		if !info.Mode().IsRegular() {
			diags = append(diags, diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "NP2004",
				Message:  fmt.Sprintf("source is not a regular file: %s", rel),
				File:     rel,
				Source:   "validate",
			})
		}
	}

	return diags
}

type sourceFile struct {
	rel  string
	abs  string
	data []byte
}

func readableSources(p project.Project, cfg config.ProjectConfig) []sourceFile {
	var files []sourceFile
	seen := map[string]bool{}
	for _, rel := range cfg.SourceFiles() {
		key := strings.ToLower(filepath.Clean(rel))
		if seen[key] {
			continue
		}
		seen[key] = true
		abs, err := p.Resolve(rel)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		files = append(files, sourceFile{rel: rel, abs: abs, data: data})
	}
	return files
}

// ---------- front matter and abstract -----------------------------------

var frontMatterRE = regexp.MustCompile(`(?s)^---[ \t]*\r?\n(.*?)\r?\n---[ \t]*(?:\r?\n|$)`)

type frontMatter struct {
	Title    string   `yaml:"title"`
	Problem  string   `yaml:"problem"`
	Keywords []string `yaml:"keywords"`
}

func validateFrontMatter(path, rel string) []diagnostic.Diagnostic {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if !utf8.Valid(data) {
		return []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "NP2105",
			Message:  "first source is not valid UTF-8",
			File:     rel,
			Source:   "validate",
		}}
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	match := frontMatterRE.FindSubmatch(data)
	if match == nil {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP2101",
			Message:    "first source must have closed YAML front matter (--- ... ---)",
			File:       rel,
			Suggestion: "Add front matter with title, problem, and keywords.",
			Source:     "validate",
		}}
	}

	var fm frontMatter
	if err := yaml.Unmarshal(match[1], &fm); err != nil {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP2101",
			Message:    fmt.Sprintf("invalid YAML front matter: %v", err),
			File:       rel,
			Suggestion: "Fix the YAML front matter.",
			Source:     "validate",
		}}
	}

	var diags []diagnostic.Diagnostic
	if strings.TrimSpace(fm.Title) == "" {
		diags = append(diags, metadataDiag("NP2102", "title is missing or empty in front matter", rel, "Add a title field."))
	}
	if strings.TrimSpace(fm.Problem) == "" {
		diags = append(diags, metadataDiag("NP2103", "problem is missing or empty in front matter", rel, "Add a problem field."))
	}
	if len(fm.Keywords) == 0 {
		diags = append(diags, metadataDiag("NP2104", "keywords are missing or empty in front matter", rel, "Add at least one keyword."))
	} else {
		for _, keyword := range fm.Keywords {
			if strings.TrimSpace(keyword) == "" {
				diags = append(diags, metadataDiag("NP2104", "keywords contain an empty value", rel, "Remove empty keywords."))
				break
			}
		}
	}
	return diags
}

func metadataDiag(code, message, file, suggestion string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityError,
		Code:       code,
		Message:    message,
		File:       file,
		Suggestion: suggestion,
		Source:     "validate",
	}
}

func validateLaterFrontMatter(files []sourceFile) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	for _, file := range files {
		data := bytes.TrimPrefix(file.data, []byte{0xEF, 0xBB, 0xBF})
		if frontMatterRE.Match(data) {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP2106",
				Message:    "only the first source may contain document front matter",
				File:       file.rel,
				Suggestion: "Remove front matter from later source files.",
				Source:     "validate",
			})
		}
	}
	return diags
}

var abstractHeadingRE = regexp.MustCompile(`(?m)^#\s+摘要\s*$`)
var levelOneHeadingRE = regexp.MustCompile(`(?m)^#\s+`)

func validateAbstract(path, rel string) []diagnostic.Diagnostic {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	matches := abstractHeadingRE.FindAllIndex(data, -1)
	if len(matches) == 0 {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP2201",
			Message:    "abstract heading (# 摘要) not found",
			File:       rel,
			Suggestion: "Add a '# 摘要' section with non-empty content.",
			Source:     "validate",
		}}
	}
	if len(matches) > 1 {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP2202",
			Message:    "multiple abstract headings (# 摘要) found",
			File:       rel,
			Suggestion: "Keep exactly one abstract section.",
			Source:     "validate",
		}}
	}

	body := data[matches[0][1]:]
	if next := levelOneHeadingRE.FindIndex(body); next != nil {
		body = body[:next[0]]
	}
	if strings.TrimSpace(string(body)) == "" {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP2203",
			Message:    "abstract content is empty",
			File:       rel,
			Suggestion: "Write abstract content below '# 摘要'.",
			Source:     "validate",
		}}
	}
	return nil
}

// ---------- resources, citations and cross references -------------------

var imageRE = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
var citationRE = regexp.MustCompile(`@([A-Za-z0-9_:.+/-]+)`)
var bibEntryRE = regexp.MustCompile(`(?m)@[A-Za-z]+\s*\{\s*([^,\s]+)\s*,`)
var crossrefIDRE = regexp.MustCompile(`\{#((?:fig|tbl|eq|sec):[A-Za-z0-9_.-]+)(?:\s+[^}]*)?\}`)
var crossrefUseRE = regexp.MustCompile(`@((?:fig|tbl|eq|sec):[A-Za-z0-9_.-]+)`)

func validateResources(p project.Project, files []sourceFile) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic

	bibPath, err := p.Resolve("references.bib")
	if err != nil {
		return diags
	}
	if info, statErr := os.Stat(bibPath); statErr != nil || !info.Mode().IsRegular() {
		diags = append(diags, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP2301",
			Message:    "references.bib not found or is not a regular file",
			File:       "references.bib",
			Suggestion: "Create a readable references.bib in the project root.",
			Source:     "validate",
		})
	}

	for _, file := range files {
		matches := imageRE.FindAllStringSubmatch(string(file.data), -1)
		for _, match := range matches {
			imagePath := strings.Trim(strings.TrimSpace(match[1]), "<>")
			if strings.Contains(imagePath, "://") {
				continue
			}
			// Product paths are rooted at Project Root, including image paths
			// referenced by Markdown files in sections/ subdirectories.
			resolved, resolveErr := p.Resolve(filepath.FromSlash(imagePath))
			if resolveErr != nil {
				diags = append(diags, diagnostic.Diagnostic{
					Severity:   diagnostic.SeverityError,
					Code:       "NP2303",
					Message:    fmt.Sprintf("image path outside project: %s", imagePath),
					File:       file.rel,
					Suggestion: "Keep image resources inside the project.",
					Source:     "validate",
				})
				continue
			}
			if info, statErr := os.Stat(resolved); statErr != nil || !info.Mode().IsRegular() {
				diags = append(diags, diagnostic.Diagnostic{
					Severity:   diagnostic.SeverityError,
					Code:       "NP2302",
					Message:    fmt.Sprintf("referenced image not found: %s", imagePath),
					File:       file.rel,
					Suggestion: "Check that the image exists at the specified path.",
					Source:     "validate",
				})
			}
		}
	}
	return diags
}

func validateCitationsAndCrossrefs(p project.Project, files []sourceFile) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	bibPath, err := p.Resolve("references.bib")
	if err != nil {
		return diags
	}
	bibData, err := os.ReadFile(bibPath)
	if err != nil {
		return diags // NP2301 already reports a missing or unreadable bibliography.
	}

	bibKeys := map[string]bool{}
	for _, match := range bibEntryRE.FindAllStringSubmatch(string(bibData), -1) {
		key := match[1]
		if bibKeys[key] {
			diags = append(diags, diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "NP3102",
				Message:  fmt.Sprintf("duplicate bibliography key: %s", key),
				File:     "references.bib",
				Source:   "validate",
			})
		}
		bibKeys[key] = true
	}

	defined := map[string]string{}
	usedCrossrefs := map[string]string{}
	usedCitations := map[string]string{}

	for _, file := range files {
		content := removeCodeBlocks(string(file.data))
		for _, match := range crossrefIDRE.FindAllStringSubmatch(content, -1) {
			id := match[1]
			if previous, ok := defined[id]; ok {
				diags = append(diags, diagnostic.Diagnostic{
					Severity:   diagnostic.SeverityError,
					Code:       "NP3201",
					Message:    fmt.Sprintf("duplicate cross-reference ID %s (already defined in %s)", id, previous),
					File:       file.rel,
					Suggestion: "Use a unique ID for every referenced element.",
					Source:     "validate",
				})
			} else {
				defined[id] = file.rel
			}
		}
		for _, match := range crossrefUseRE.FindAllStringSubmatch(content, -1) {
			usedCrossrefs[match[1]] = file.rel
		}
		for _, match := range citationRE.FindAllStringSubmatch(content, -1) {
			key := match[1]
			if isCrossrefKey(key) {
				continue
			}
			usedCitations[key] = file.rel
		}
	}

	for _, key := range sortedKeys(usedCitations) {
		if !bibKeys[key] {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP3101",
				Message:    fmt.Sprintf("citation key not found in references.bib: %s", key),
				File:       usedCitations[key],
				Suggestion: "Add the key to references.bib or fix the citation.",
				Source:     "validate",
			})
		}
	}
	for _, id := range sortedKeys(usedCrossrefs) {
		if _, ok := defined[id]; !ok {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP3202",
				Message:    fmt.Sprintf("cross-reference target not found: %s", id),
				File:       usedCrossrefs[id],
				Suggestion: "Define the referenced figure, table, equation, or section ID.",
				Source:     "validate",
			})
		}
	}
	return diags
}

func isCrossrefKey(key string) bool {
	for _, prefix := range []string{"fig:", "tbl:", "eq:", "sec:"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ---------- controlled LaTeX fragment use -------------------------------

var fragmentInputRE = regexp.MustCompile(`\\input\s*\{([^}\r\n]+)\}`)
var unsafeRawTeXCommandRE = regexp.MustCompile(`(?i)\\(?:include|includeonly|inputiffileexists|subfile|import|subimport|includegraphics|verbatiminput|lstinputlisting|bibliography|addbibresource|write18|shellescape|pdfshellescape|immediate|openin|openout|read|write|catcode|csname|scantokens|special|directlua|endlinechar|escapechar)\b`)

func validateDeclaredFragmentInputs(files []sourceFile, cfg config.ProjectConfig) []diagnostic.Diagnostic {
	declared := make(map[string]bool, len(cfg.LatexFragments))
	for _, path := range cfg.LatexFragments {
		declared[strings.ToLower(filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))))] = true
	}
	var diags []diagnostic.Diagnostic
	for _, file := range files {
		content := removeCodeBlocks(string(file.data))
		for _, match := range fragmentInputRE.FindAllStringSubmatchIndex(content, -1) {
			path := strings.TrimSpace(content[match[2]:match[3]])
			key := strings.ToLower(filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))))
			if declared[key] {
				continue
			}
			line := strings.Count(content[:match[0]], "\n") + 1
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       fragment.CodeUndeclaredInput,
				Message:    fmt.Sprintf("LaTeX fragment input is not declared: %s", path),
				File:       file.rel,
				Line:       line,
				Suggestion: "Add the project-relative .tex path to latexFragments or remove the input.",
				Source:     "validate",
			})
		}
		collapsed := collapseTeXComments(content)
		if location := unsafeRawTeXCommandRE.FindStringIndex(collapsed); location != nil {
			diags = append(diags, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       fragment.CodeCommandExecution,
				Message:    fmt.Sprintf("unsafe raw TeX command is not allowed: %s", unsafeRawTeXCommandRE.FindString(collapsed)),
				File:       file.rel,
				Suggestion: "Use only explicitly declared \\input{...} Fragments for advanced table or equation layout.",
				Source:     "validate",
			})
		}
	}
	return diags
}

func collapseTeXComments(content string) string {
	var result strings.Builder
	for _, line := range strings.Split(content, "\n") {
		stripped := stripRawTeXComment(line)
		result.WriteString(stripped)
		if len(stripped) == len(line) {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

func stripRawTeXComment(line string) string {
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

// ---------- raw LaTeX detection -----------------------------------------

var rawLatexPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\\begin\{`),
	regexp.MustCompile(`\\def\b`),
	regexp.MustCompile(`\\newcommand\b`),
	regexp.MustCompile(`\\renewcommand\b`),
	regexp.MustCompile(`\\usepackage\b`),
}

func validateRawLatex(files []sourceFile) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	for _, file := range files {
		content := removeCodeAndMath(string(file.data))
		for _, pattern := range rawLatexPatterns {
			if location := pattern.FindStringIndex(content); location != nil {
				line := strings.Count(content[:location[0]], "\n") + 1
				diags = append(diags, diagnostic.Diagnostic{
					Severity:   diagnostic.SeverityWarning,
					Code:       "NP2401",
					Message:    fmt.Sprintf("raw LaTeX found near line %d: %s", line, pattern.FindString(content)),
					File:       file.rel,
					Line:       line,
					Suggestion: "Raw LaTeX is only guaranteed for PDF output.",
					Source:     "validate",
				})
				break
			}
		}
	}
	return diags
}

func removeCodeBlocks(content string) string {
	fenced := regexp.MustCompile("(?s)```.*?```")
	content = fenced.ReplaceAllString(content, "")
	inline := regexp.MustCompile("`[^`\\n]+`")
	return inline.ReplaceAllString(content, "")
}

func removeCodeAndMath(content string) string {
	content = removeCodeBlocks(content)
	blockMath := regexp.MustCompile(`(?s)\$\$.*?\$\$`)
	content = blockMath.ReplaceAllString(content, "")
	inlineMath := regexp.MustCompile(`\$[^$\n]+\$`)
	return inlineMath.ReplaceAllString(content, "")
}

// ---------- config diagnostics ------------------------------------------

func configDiag(err error) diagnostic.Diagnostic {
	code := "NP1502"
	if strings.Contains(err.Error(), "cannot parse YAML") {
		code = "NP1501"
	}
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityError,
		Code:       code,
		Message:    fmt.Sprintf("invalid configuration: %v", err),
		File:       "nodepaper.yaml",
		Suggestion: "Fix nodepaper.yaml and try again.",
		Source:     "validate",
	}
}
