package validate

import (
	"strings"
	"testing"
)

// citationKeys mirrors what validateCitationsAndCrossrefs collects, so these
// cases pin the scanning rules without needing a Project on disk.
func citationKeys(t *testing.T, markdown string) []string {
	t.Helper()
	content := removeCodeAndMath(markdown)
	var keys []string
	for _, match := range citationRE.FindAllStringSubmatch(content, -1) {
		if isCrossrefKey(match[1]) {
			continue
		}
		keys = append(keys, match[1])
	}
	return keys
}

func assertNoKeys(t *testing.T, markdown, why string) {
	t.Helper()
	if keys := citationKeys(t, markdown); len(keys) != 0 {
		t.Fatalf("%s: got citation keys %v, want none\n---\n%s", why, keys, markdown)
	}
}

// amscd draws arrows with @VVgV and @>>>. Reading those as citation keys made
// valid mathematics fail the build with NP3101.
func TestCitationScanIgnoresDisplayMath(t *testing.T) {
	assertNoKeys(t, `正文。

$$
\begin{CD}
A @>f>> B \\
@VVgV @VVhV \\
C @>k>> D
\end{CD}
$$
`, "amscd arrows inside a display equation")

	assertNoKeys(t, "行内数学 $x @VVgV y$ 也一样。\n", "inline math")
}

func TestCitationScanIgnoresCode(t *testing.T) {
	assertNoKeys(t, "行内代码 `@media screen` 不是引用。\n", "inline code")
	assertNoKeys(t, "```css\n@media (min-width: 600px) { body { color: red } }\n```\n", "fenced code")
}

// An address in prose is not a citation; the key "example" used to be reported
// as a missing reference.
func TestCitationScanIgnoresEmailAddresses(t *testing.T) {
	assertNoKeys(t, "联系 foo@example 或 user.name@domain 均可。\n", "email addresses in prose")
}

// The guard against emails must not cost us the forms authors actually use.
func TestCitationScanStillFindsRealCitations(t *testing.T) {
	for _, tc := range []struct {
		name     string
		markdown string
		want     string
	}{
		{"line start", "@zhang2020 指出……\n", "zhang2020"},
		{"after space", "如 @zhang2020 所述。\n", "zhang2020"},
		{"bracketed", "结论一致[@zhang2020]。\n", "zhang2020"},
		{"after Chinese text and space", "见 @zhang2020 的讨论。\n", "zhang2020"},
		{"after an opening paren", "(@zhang2020)\n", "zhang2020"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keys := citationKeys(t, tc.markdown)
			if len(keys) != 1 || keys[0] != tc.want {
				t.Fatalf("keys = %v, want [%s]", keys, tc.want)
			}
		})
	}
}

func TestCitationScanSeparatesMultipleKeys(t *testing.T) {
	keys := citationKeys(t, "[@a2020;@b2021] 与 @c2022。\n")
	joined := strings.Join(keys, ",")
	if joined != "a2020,b2021,c2022" {
		t.Fatalf("keys = %v, want a2020 b2021 c2022", keys)
	}
}

// Crossref IDs are handled by their own list and must not be double-counted as
// citations.
func TestCitationScanSkipsCrossrefKeys(t *testing.T) {
	assertNoKeys(t, "如 @fig:overview 与 @tbl:yield 所示。\n", "crossref references")
}
