// Package doctor inspects the local environment and reports whether the
// NodePaper toolchain is ready to build.
package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nodepaper/internal/diagnostic"
	"nodepaper/internal/fonts"
	"nodepaper/internal/latexlog"
	"nodepaper/internal/process"
	"nodepaper/internal/profile"
)

// Status summarises a single tool or resource check.
type Status string

const (
	StatusPass    Status = "pass"
	StatusWarning Status = "warning"
	StatusFail    Status = "fail"
	StatusSkipped Status = "skipped"
)

// Check is the outcome of inspecting a single tool or resource.
type Check struct {
	Name       string `json:"name"`
	Status     Status `json:"status"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Result collects the findings from a doctor run.
type Result struct {
	Success     bool                    `json:"success"`
	ProjectRoot string                  `json:"projectRoot,omitempty"`
	Checks      []Check                 `json:"checks"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

// Toolchain represents the local tool binaries. Paths may be empty when a tool
// is not installed.
type Toolchain struct {
	Powershell     string
	Pandoc         string
	PandocCrossref string
	Latexmk        string
	XeLaTeX        string
}

// Run inspects the local environment. resourceRoot is the directory that
// contains the built-in profiles/ and tools/ trees next to the executable; it
// is empty when the packaged resources cannot be located. When projectRoot is
// non-empty it also verifies project resources.
func Run(ctx context.Context, projectRoot string, resourceRoot string) Result {
	var checks []Check

	profileDir := ""
	if resourceRoot != "" {
		profileDir = filepath.Join(resourceRoot, "profiles", "cumcm")
	}
	loadedProfile, profileCheck := checkBuiltinProfile(profileDir)
	checks = append(checks, profileCheck)
	checks = append(checks, checkPowershell(ctx))
	tc := findToolchain(resourceRoot)
	checks = append(checks,
		checkPandoc(ctx, tc.Pandoc, loadedProfile.Definition.PandocVersion),
		checkPandocCrossref(ctx, tc.PandocCrossref, loadedProfile.Definition.PandocCrossrefVersion),
		checkXeLaTeXDriver(ctx, tc),
		checkXeLaTeX(ctx, tc.XeLaTeX),
	)
	checks = append(checks, checkChineseProbe(ctx, tc))
	checks = append(checks, checkLaTeXExportPackages(ctx))

	if projectRoot != "" {
		checks = append(checks, checkProjectResources(projectRoot)...)
	}

	result := Result{Success: true, Checks: checks}
	for _, c := range checks {
		if c.Status == StatusFail {
			result.Success = false
			result.Diagnostics = append(result.Diagnostics, checkToDiag(c))
		}
		if c.Status == StatusWarning {
			result.Diagnostics = append(result.Diagnostics, checkToDiag(c))
		}
	}
	if projectRoot != "" {
		result.ProjectRoot = projectRoot
	}
	return result
}

// ---------- individual checks -------------------------------------------

func checkBuiltinProfile(dir string) (profile.Loaded, Check) {
	if dir == "" {
		return profile.Loaded{}, Check{
			Name:       "profile",
			Status:     StatusFail,
			Message:    "built-in CUMCM Profile directory was not found",
			Suggestion: "Reinstall NodePaper with profiles/cumcm next to nodepaper.exe.",
		}
	}
	loaded, err := profile.Load(dir)
	if err != nil {
		return profile.Loaded{}, Check{
			Name:       "profile",
			Status:     StatusFail,
			Message:    fmt.Sprintf("CUMCM Profile is invalid: %v", err),
			Suggestion: "Reinstall NodePaper or restore profiles/cumcm.",
		}
	}
	return loaded, Check{
		Name:    "profile",
		Status:  StatusPass,
		Message: fmt.Sprintf("%s (rules %s, version %s, sha256 %s)", loaded.Dir, loaded.Definition.RulesVersion, loaded.Definition.Version, loaded.SHA256),
	}
}

func checkPowershell(ctx context.Context) Check {
	path, err := exec.LookPath("powershell.exe")
	if err != nil {
		path, err = exec.LookPath("pwsh")
	}
	if err != nil {
		return Check{
			Name:       "PowerShell",
			Status:     StatusFail,
			Message:    "PowerShell not found",
			Suggestion: "Install PowerShell or ensure powershell.exe/pwsh is in PATH.",
		}
	}
	version, err := commandVersion(ctx, path, []string{"-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()"}, regexp.MustCompile(`^\d+(?:\.\d+)+$`))
	if err != nil {
		return Check{Name: "PowerShell", Status: StatusFail, Message: err.Error(), Suggestion: "Repair PowerShell and run doctor again."}
	}
	return Check{Name: "PowerShell", Status: StatusPass, Message: fmt.Sprintf("%s (%s)", path, version)}
}

// findToolchain returns the resolved tool binaries. The pinned bundled
// binaries shipped in the release package take precedence over PATH entries so
// that doctor truthfully reports what the build will actually use on machines
// that do not install Pandoc or pandoc-crossref globally.
func findToolchain(resourceRoot string) Toolchain {
	var tc Toolchain
	tc.Pandoc = bundledOrPath(resourceRoot, "pandoc", "pandoc.exe", "pandoc")
	tc.PandocCrossref = bundledOrPath(resourceRoot, "pandoc-crossref", "pandoc-crossref.exe", "pandoc-crossref")
	tc.Latexmk, _ = exec.LookPath("latexmk")
	tc.XeLaTeX, _ = exec.LookPath("xelatex")
	return tc
}

// bundledOrPath prefers the pinned binary bundled under
// <resourceRoot>/tools/windows-x64/<toolDir>/<exeName> and falls back to a
// command found on PATH.
func bundledOrPath(resourceRoot, toolDir, exeName, command string) string {
	if resourceRoot != "" {
		path := filepath.Join(resourceRoot, "tools", "windows-x64", toolDir, exeName)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	if path, err := exec.LookPath(command); err == nil {
		return path
	}
	return ""
}

func checkPandoc(ctx context.Context, path, expected string) Check {
	if path == "" {
		return missingTool("Pandoc", "Reinstall NodePaper (the bundled pandoc is missing) or install the pinned Pandoc version and ensure it is in PATH.")
	}
	version, err := commandVersion(ctx, path, []string{"--version"}, regexp.MustCompile(`^pandoc\s+\S+`))
	if err != nil {
		return failedVersionCheck("Pandoc", err)
	}
	if expected != "" && version != "pandoc "+expected {
		return Check{Name: "Pandoc", Status: StatusFail, Message: fmt.Sprintf("expected pandoc %s, got %s", expected, version), Suggestion: "Install the Profile-pinned Pandoc version."}
	}
	return Check{Name: "Pandoc", Status: StatusPass, Message: fmt.Sprintf("%s (%s)", path, version)}
}

func checkPandocCrossref(ctx context.Context, path, expected string) Check {
	if path == "" {
		return missingTool("pandoc-crossref", "Reinstall NodePaper (the bundled pandoc-crossref is missing) or install the pinned pandoc-crossref version and ensure it is in PATH.")
	}
	version, err := commandVersion(ctx, path, []string{"--version"}, regexp.MustCompile(`^pandoc-crossref\s+v?\S+`))
	if err != nil {
		return failedVersionCheck("pandoc-crossref", err)
	}
	actual := strings.TrimPrefix(strings.Fields(version)[1], "v")
	if expected != "" && actual != expected {
		return Check{Name: "pandoc-crossref", Status: StatusFail, Message: fmt.Sprintf("expected %s, got %s", expected, actual), Suggestion: "Install the Profile-pinned pandoc-crossref version."}
	}
	return Check{Name: "pandoc-crossref", Status: StatusPass, Message: fmt.Sprintf("%s (%s)", path, version)}
}

// checkXeLaTeXDriver reports how the PDF stage is driven. NodePaper runs
// XeLaTeX directly and repeats the pass until cross-references stabilise, so
// latexmk - a Perl script that MiKTeX cannot run without a separate Perl
// install - is no longer required. It is still reported when present because
// users often expect to see it.
func checkXeLaTeXDriver(ctx context.Context, tc Toolchain) Check {
	if tc.Latexmk == "" {
		return Check{
			Name:    "LaTeX driver",
			Status:  StatusPass,
			Message: "NodePaper drives XeLaTeX directly (latexmk and Perl are not required)",
		}
	}
	version, err := commandVersion(ctx, tc.Latexmk, []string{"-v"}, regexp.MustCompile(`^Latexmk,.*Version\s+\S+`))
	if err != nil {
		// A latexmk that cannot report its version is almost always a MiKTeX
		// installation without Perl. It no longer blocks the build.
		return Check{
			Name:       "LaTeX driver",
			Status:     StatusPass,
			Message:    "NodePaper drives XeLaTeX directly; the latexmk found on PATH cannot run (typically MiKTeX without Perl) and is unused",
			Suggestion: "No action needed for NodePaper. Install Strawberry Perl only if other tools of yours need latexmk.",
		}
	}
	return Check{Name: "LaTeX driver", Status: StatusPass, Message: fmt.Sprintf("XeLaTeX driven directly; latexmk also available (%s)", version)}
}

// texDistributionHelp is the whole answer to "XeLaTeX not found", because the
// one-line version of it was not actionable: NodePaper installs in seconds
// while the TeX distribution behind it is the multi-gigabyte, multi-hour part,
// and a user who is not told that has no way to plan for it. Sizes are from
// 2026-08-15 (basic-miktex-25.12-x64.exe, texlive2026-20260301.iso); installed
// footprint and duration are order-of-magnitude figures.
const texDistributionHelp = `NodePaper does not bundle TeX; a TeX distribution is required.
  MiKTeX     ~140 MB download, ~1 GB installed, 10-20 min
             https://miktex.org/download
  TeX Live   ~6.3 GB download, ~8-9 GB installed, 20-60 min with a local mirror
             https://tug.org/texlive/windows.html
Install to a path without spaces or non-ASCII characters. Then open a NEW terminal
(PATH changes do not affect already-open windows), run ` + "`xelatex --version`" + `, and
re-run ` + "`nodepaper doctor`" + `.
NodePaper does not require latexmk or Perl.`

// xelatexVersionLine matches the line on which a XeLaTeX binary states its
// identity. It looks for XeTeX *anywhere* on the line rather than at the start,
// because only TeX Live opens with a bare
//
//	XeTeX 3.141592653-2.6-0.999997 (TeX Live 2025)
//
// MiKTeX prefixes its own name instead, so an anchored `^XeTeX` pattern matched
// no line at all there and doctor reported "cannot identify xelatex.exe version
// output" - a hard failure on the very distribution NodePaper recommends
// (~140 MB against TeX Live's ~6.3 GB). This check only has to confirm that
// xelatex runs and names itself; parsing one distribution's exact layout is not
// its job.
//
// The trailing digit is what keeps the relaxation honest. TeX Live's own
// --version output repeats "XeTeX" in later prose lines ("covered by the terms
// of both the XeTeX copyright and", "Primary author of XeTeX: Jonathan Kew."),
// and none of those carry a number, so requiring a digit after the name selects
// version lines over prose. commandVersion returns the first matching line and
// every known distribution states its identity on line 1, so this is a
// second line of defence rather than the load-bearing part.
//
// Confirmed against a real MiKTeX on 2026-08-17 (miktex-e2e run 31993318613),
// which prints:
//
//	MiKTeX-XeTeX 4.18 (MiKTeX 26.5)
//
// The line above it on that run is "xelatex: security risk: running with
// elevated privileges", written to stderr and folded in by CombinedOutput; it
// says "xelatex" in lower case and so cannot be mistaken for the version line.
var xelatexVersionLine = regexp.MustCompile(`XeTeX\b[^0-9]*\d`)

func checkXeLaTeX(ctx context.Context, path string) Check {
	if path == "" {
		return Check{
			Name:       "XeLaTeX",
			Status:     StatusFail,
			Message:    "XeLaTeX not found",
			Suggestion: texDistributionHelp,
		}
	}
	version, err := commandVersion(ctx, path, []string{"--version"}, xelatexVersionLine)
	if err != nil {
		return failedVersionCheck("XeLaTeX", err)
	}
	return Check{Name: "XeLaTeX", Status: StatusPass, Message: fmt.Sprintf("%s (%s)", path, version)}
}

// latexExportPackagesCheckName is the fixed Name for the check below. It is
// suffixed "(optional)" so the label itself signals, before Message or
// Suggestion are read, that this is not part of the core build path.
const latexExportPackagesCheckName = "LaTeX export packages (optional)"

// checkLaTeXExportPackages probes for the packages used only by
// `nodepaper export` (gbt7714, biblatex-gb7714-2015, biber). The core build
// path (`nodepaper build`) never touches them, and most users never run
// export, so this check always reports StatusPass regardless of what it
// finds: their absence is a normal, unremarkable state, not a warning. See
// checkToDiag (only Warning/Fail become diagnostics) - Pass here is what
// keeps this check out of the Warning count.
func checkLaTeXExportPackages(ctx context.Context) Check {
	kpsewhichPath, err := exec.LookPath("kpsewhich")
	if err != nil {
		return Check{
			Name:   latexExportPackagesCheckName,
			Status: StatusPass,
			Message: "kpsewhich is unavailable, so gbt7714 / biblatex-gb7714-2015 / biber " +
				"were not probed; these are used only by nodepaper export and never by nodepaper build",
		}
	}

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	gbt7714 := kpsewhichHasFile(probeCtx, kpsewhichPath, "gbt7714.sty")
	biblatexGB := kpsewhichHasFile(probeCtx, kpsewhichPath, "gb7714-2015.bbx")
	_, err = exec.LookPath("biber")
	biber := err == nil

	return latexExportPackagesResult(gbt7714, biblatexGB, biber)
}

// latexExportPackagesResult renders the probe outcome into a Check. It is
// kept separate from checkLaTeXExportPackages so the three combinations
// (all present / all absent / partially present) can be exercised directly
// in tests without shelling out.
func latexExportPackagesResult(gbt7714, biblatexGB, biber bool) Check {
	state := func(found bool) string {
		if found {
			return "available"
		}
		return "not detected"
	}

	message := fmt.Sprintf(
		"gbt7714 %s; biblatex-gb7714-2015 %s; biber %s (used only by the optional nodepaper export)",
		state(gbt7714), state(biblatexGB), state(biber),
	)

	check := Check{Name: latexExportPackagesCheckName, Status: StatusPass, Message: message}
	if gbt7714 && biblatexGB && biber {
		return check
	}

	check.Suggestion = "If you would like to export a LaTeX project with nodepaper export, " +
		"you can optionally install these:\n" +
		"  tlmgr install gbt7714 biblatex-gb7714-2015 biber\n" +
		"  miktex packages install gbt7714 biblatex-gb7714-2015 biber"
	return check
}

// kpsewhichHasFile reports whether kpsewhich resolves filename to a path. A
// non-zero exit status or empty output both mean "not found" and are treated
// the same way: as a plain false, never surfaced as an error.
func kpsewhichHasFile(ctx context.Context, kpsewhichPath, filename string) bool {
	output, err := exec.CommandContext(ctx, kpsewhichPath, filename).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func missingTool(name, suggestion string) Check {
	return Check{Name: name, Status: StatusFail, Message: name + " not found", Suggestion: suggestion}
}

func failedVersionCheck(name string, err error) Check {
	return Check{Name: name, Status: StatusFail, Message: err.Error(), Suggestion: "Verify that the executable can run and report its version."}
}

func commandVersion(ctx context.Context, path string, args []string, linePattern *regexp.Regexp) (string, error) {
	command := exec.CommandContext(ctx, path, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot read %s version: %w", filepath.Base(path), err)
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if linePattern.MatchString(line) {
			return line, nil
		}
	}
	return "", fmt.Errorf("cannot identify %s version output", filepath.Base(path))
}

// chineseProbeStandard is what NodePaper builds when SimHei and KaiTi are
// installed: a bare ctexart, whose default fontset=windows binds
// SimSun/SimHei/KaiTi by itself.
const chineseProbeStandard = `\documentclass[UTF8]{ctexart}
\begin{document}
NodePaper 中文环境探针
\end{document}
`

// chineseProbeFallback is what NodePaper builds when either supplemental font
// is absent. Its preamble is the fallback branch of profiles/cumcm/template.tex
// (the $if(nodepaper-font-fallback)$ arm) reproduced line for line, minus the
// zihao/a4paper options that only affect page layout: fontset=none so ctex
// binds nothing, every family bound to SimSun, the missing weights synthesised,
// and the \songti-family commands restored because fontset=none does not define
// them.
//
// Keeping these in step with the profile is the whole point of the check. A
// probe that compiles something the build never compiles can only produce two
// wrong answers, and doctor had been giving the worse one: on a machine without
// SimHei the bare ctexart above fails while `nodepaper build` succeeds through
// exactly this fallback, so doctor condemned an environment that works.
const chineseProbeFallback = `\documentclass[UTF8,fontset=none]{ctexart}
\setCJKmainfont{SimSun}[AutoFakeBold=true, AutoFakeSlant=true]
\setCJKfamilyfont{zhsong}{SimSun}[AutoFakeBold=true, AutoFakeSlant=true]
\setCJKfamilyfont{zhhei}{SimSun}[AutoFakeBold=true]
\setCJKfamilyfont{zhkai}{SimSun}[AutoFakeSlant=true]
\setCJKfamilyfont{zhfs}{SimSun}
\providecommand{\songti}{\CJKfamily{zhsong}}
\providecommand{\heiti}{\CJKfamily{zhhei}}
\providecommand{\kaishu}{\CJKfamily{zhkai}}
\providecommand{\fangsong}{\CJKfamily{zhfs}}
\begin{document}
NodePaper 中文环境探针
\end{document}
`

// chineseProbeDocument picks the probe source the way the build picks its
// preamble, and reports whether that was the fallback.
//
// Anything short of "both fonts positively found" selects the fallback,
// undetermined included. That is the build's rule as well: the font probe in
// Convert-CumcmProjectToLatex.ps1 sets nodepaper-font-fallback whenever
// Test-Path fails to find the file, for whatever reason. The asymmetry is
// deliberate - the fallback preamble also compiles on a machine that does have
// SimHei and KaiTi, whereas the bare ctexart does not compile without them.
func chineseProbeDocument(availability fonts.Availability) (document string, usedFallback bool) {
	if availability.AllPresent() {
		return chineseProbeStandard, false
	}
	return chineseProbeFallback, true
}

// chineseProbeFailureSuggestion explains a failed probe in terms of the
// document that actually failed. Under the fallback the old advice pointed at
// the wrong thing: the optional fonts are already known to be absent and are
// not what the probe used, so what remains is SimSun itself or an incomplete
// ctex/xeCJK installation.
func chineseProbeFailureSuggestion(usedFallback bool) string {
	if !usedFallback {
		return "Inspect the TeX installation, ctex package and configured Chinese fonts."
	}
	return "SimHei and KaiTi are not installed, so this probe compiled NodePaper's " +
		"SimSun-only fallback preamble - the same one a build uses. The optional fonts " +
		"are therefore not the cause. Check that SimSun (simsun.ttc) is present in the " +
		"Windows font directory and that ctex and xeCJK are fully installed " +
		"(miktex packages install ctex xecjk, or tlmgr install ctex xecjk)."
}

func checkChineseProbe(ctx context.Context, tc Toolchain) Check {
	if tc.XeLaTeX == "" {
		return Check{
			Name:       "Chinese TeX probe",
			Status:     StatusSkipped,
			Message:    "XeLaTeX prerequisite is missing",
			Suggestion: "Install TeX Live or MiKTeX, then run doctor again.",
		}
	}

	dir, err := os.MkdirTemp("", "nodepaper-doctor-probe-*")
	if err != nil {
		return Check{Name: "Chinese TeX probe", Status: StatusFail, Message: fmt.Sprintf("cannot create probe directory: %v", err)}
	}
	defer os.RemoveAll(dir)

	document, usedFallback := chineseProbeDocument(fonts.ProbeSupplemental())
	texPath := filepath.Join(dir, "probe.tex")
	if err := os.WriteFile(texPath, []byte(document), 0o644); err != nil {
		return Check{Name: "Chinese TeX probe", Status: StatusFail, Message: fmt.Sprintf("cannot write probe: %v", err)}
	}

	// The probe runs in its own directory and names the file relatively, never
	// as an absolute path. TeX reads the filename argument as TeX tokens, so a
	// "~" in it is the active character, not a literal: given
	// C:\Users\RUNNER~1\AppData\Local\Temp\... XeLaTeX stops early and reports
	//
	//	I can't find file `C:/Users/RUNNER'.
	//
	// That is not a CI quirk. Windows generates such 8.3 names for any account
	// whose name exceeds eight characters, and %TEMP% sits under the profile,
	// so ordinary users hit it. Keeping the path out of the argument also
	// sidesteps spaces and non-ASCII characters. Diagnosed on miktex-e2e run
	// 31994977105, where a hand-run of the same document under a long path
	// compiled cleanly.
	runner := &process.Runner{Dir: dir, CaptureSize: 256 * 1024}
	processResult, runErr := runner.Run(ctx, tc.XeLaTeX,
		"-interaction=nonstopmode",
		"-halt-on-error",
		"-file-line-error",
		"-output-directory=.",
		filepath.Base(texPath),
	)
	if runErr != nil || processResult.ExitCode != 0 {
		message := fmt.Sprintf("minimal Chinese document failed with exit code %d", processResult.ExitCode)
		if runErr != nil {
			message = fmt.Sprintf("minimal Chinese document failed: %v", runErr)
		}
		// An exit code alone leaves the reader guessing: a missing .sty, an
		// unresolvable font and a broken installation all look identical and
		// need different fixes. XeLaTeX prints the error to stdout as well as
		// to the log, and the captured stream is there whether or not the log
		// was ever written -- on miktex-e2e run 31994564628 the log yielded
		// nothing while the process had failed with a real error. Read the
		// streams first and fall back to the log.
		if reason := firstLaTeXError(processResult.Stdout, processResult.Stderr, readFileString(filepath.Join(dir, "probe.log"))); reason != "" {
			message = fmt.Sprintf("%s: %s", message, reason)
		}
		return Check{
			Name:       "Chinese TeX probe",
			Status:     StatusFail,
			Message:    message,
			Suggestion: chineseProbeFailureSuggestion(usedFallback),
		}
	}

	pdfPath := filepath.Join(dir, "probe.pdf")
	data, err := os.ReadFile(pdfPath)
	if err != nil || len(data) < 5 || string(data[:5]) != "%PDF-" {
		return Check{
			Name:       "Chinese TeX probe",
			Status:     StatusFail,
			Message:    "minimal Chinese compile did not produce a readable PDF",
			Suggestion: "Inspect the TeX log and reinstall the ctex/XeLaTeX components.",
		}
	}
	return chineseProbeResult(filepath.Join(dir, "probe.log"))
}

// chineseProbeResult turns the probe's own log into a per-font report. The
// families are reported individually because they do not fail together: SimSun
// is a core Windows font, while SimHei and KaiTi ship as the optional "Chinese
// (Simplified) Supplemental Fonts" feature and are routinely absent on a
// Windows install that never added Chinese. A single "Chinese fonts OK/not OK"
// line cannot tell the user which of those situations they are in, and so
// cannot tell them what to do about it.
func chineseProbeResult(logPath string) Check {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return Check{
			Name:    "Chinese TeX probe",
			Status:  StatusPass,
			Message: "minimal ctex document compiled successfully",
		}
	}

	families := latexlog.Fonts(data)
	if len(families) == 0 {
		return Check{
			Name:    "Chinese TeX probe",
			Status:  StatusPass,
			Message: "minimal ctex document compiled successfully",
		}
	}

	names := make([]string, 0, len(families))
	synthesised := false
	for _, family := range families {
		names = append(names, family.Font)
		if strings.Contains(family.Options, "AutoFake") {
			synthesised = true
		}
	}
	resolved := strings.Join(dedupeStrings(names), ", ")

	if synthesised {
		return Check{
			Name:    "Chinese TeX probe",
			Status:  StatusWarning,
			Message: fmt.Sprintf("compiled successfully using %s, with styles synthesised", resolved),
			Suggestion: "Builds still succeed and no characters are lost, but bold and italic are " +
				"synthesised rather than real faces. To install the real ones, open Settings > " +
				"System > Optional features > Add an optional feature and pick " +
				"\"Chinese (Simplified) Supplemental Fonts\".",
		}
	}
	return Check{
		Name:    "Chinese TeX probe",
		Status:  StatusPass,
		Message: fmt.Sprintf("minimal ctex document compiled successfully using %s", resolved),
	}
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func checkProjectResources(projectRoot string) []Check {
	var checks []Check

	// Check nodepaper.yaml exists.
	configPath := filepath.Join(projectRoot, "nodepaper.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		checks = append(checks, Check{
			Name:       "nodepaper.yaml",
			Status:     StatusFail,
			Message:    fmt.Sprintf("Project config not found: %s", configPath),
			Suggestion: "Run 'nodepaper init' to create a project.",
		})
	} else {
		checks = append(checks, Check{
			Name:    "nodepaper.yaml",
			Status:  StatusPass,
			Message: configPath,
		})
	}

	return checks
}

// ---------- helpers ------------------------------------------------------

func checkToDiag(c Check) diagnostic.Diagnostic {
	severity := diagnostic.SeverityError
	if c.Status == StatusWarning || c.Status == StatusSkipped {
		severity = diagnostic.SeverityWarning
	}
	return diagnostic.Diagnostic{
		Severity:   severity,
		Code:       fmt.Sprintf("NP4%02d", severityCode(c)),
		Message:    fmt.Sprintf("[%s] %s: %s", c.Status, c.Name, c.Message),
		Suggestion: c.Suggestion,
		Source:     "doctor",
	}
}

func severityCode(c Check) int {
	switch c.Status {
	case StatusPass:
		return 0
	case StatusWarning:
		return 1
	case StatusFail:
		return 2
	default:
		return 9
	}
}

// FormatChecks returns a multi-line human-readable summary of checks.
func FormatChecks(checks []Check) string {
	var sb strings.Builder
	for _, c := range checks {
		prefix := "PASS"
		switch c.Status {
		case StatusWarning:
			prefix = "WARN"
		case StatusFail:
			prefix = "FAIL"
		case StatusSkipped:
			prefix = "SKIP"
		}
		fmt.Fprintf(&sb, "%-4s %-20s %s\n", prefix, c.Name, c.Message)
		if c.Suggestion != "" {
			fmt.Fprintf(&sb, "     → %s\n", c.Suggestion)
		}
	}
	return sb.String()
}

// firstLaTeXError returns the first error line found across the given sources,
// or "" when none carries one. TeX writes errors either as "! message" or,
// under -file-line-error, as "file:line: ! message"; both forms are reduced to
// the message so the caller can put it on one line. Sources are searched in
// order, so pass the most reliable one first.
func firstLaTeXError(sources ...string) string {
	for _, source := range sources {
		for _, line := range strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "! ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "!"))
			}
			if index := strings.Index(line, ": ! "); index > 0 {
				return strings.TrimSpace(line[index+3:])
			}
		}
	}
	return ""
}

// readFileString returns a file's contents, or "" when it cannot be read. The
// caller treats an unreadable file as simply having nothing to contribute.
func readFileString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
