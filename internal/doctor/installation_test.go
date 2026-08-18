package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNodePaperDirsOnPathKeepsResolutionOrder(t *testing.T) {
	separator := string(os.PathListSeparator)
	first := filepath.Join("root", "first")
	second := filepath.Join("root", "second")
	empty := filepath.Join("root", "empty")

	// Quoted, repeated, blank and trailing-separator entries all occur in a
	// real user PATH; none of them may change the resolution order or produce
	// a duplicate.
	value := strings.Join([]string{
		"",
		"  ",
		`"` + first + `"`,
		empty,
		first + string(filepath.Separator),
		second,
	}, separator)

	exists := func(path string) bool {
		dir := normalizeDirectory(filepath.Dir(path))
		return filepath.Base(path) == nodePaperExeName &&
			(dir == normalizeDirectory(first) || dir == normalizeDirectory(second))
	}

	dirs := nodePaperDirsOnPath(value, exists)
	if len(dirs) != 2 {
		t.Fatalf("dirs = %#v, want two entries", dirs)
	}
	if !sameDirectory(dirs[0], first) {
		t.Fatalf("dirs[0] = %q, want %q first (PATH order decides)", dirs[0], first)
	}
	if !sameDirectory(dirs[1], second) {
		t.Fatalf("dirs[1] = %q, want %q", dirs[1], second)
	}
}

func TestSameDirectoryIgnoresTrailingSeparatorAndCase(t *testing.T) {
	dir := filepath.Join("root", "NodePaper")
	if !sameDirectory(dir, strings.ToLower(dir)+string(filepath.Separator)) {
		t.Fatalf("%q and its lowercase trailing-separator form must compare equal", dir)
	}
	if !sameDirectory(dir, `"`+dir+`"`) {
		t.Fatalf("a quoted entry must compare equal to the same directory")
	}
	if sameDirectory(dir, "") || sameDirectory("", dir) {
		t.Fatalf("an unknown directory must never compare equal")
	}
	if sameDirectory(dir, filepath.Join("root", "Other")) {
		t.Fatalf("different directories must not compare equal")
	}
}

// A directory only counts as a portable release when it carries neither of
// Setup's marks. Both marks have to exclude it: a folder Setup installed into
// holds its uninstaller, and a folder Setup's uninstall entry names is its own
// even if that uninstaller is missing.
func TestPortableDirsAmongExcludesSetupDirectories(t *testing.T) {
	portable := filepath.Join("root", "portable")
	markedByUninstaller := filepath.Join("root", "taken-over")
	markedByRegistry := filepath.Join("programs", "NodePaper")

	exists := func(path string) bool {
		return filepath.Base(path) == setupUninstallerName &&
			normalizeDirectory(filepath.Dir(path)) == normalizeDirectory(markedByUninstaller)
	}

	dirs := portableDirsAmong(
		[]string{markedByUninstaller, portable, markedByRegistry},
		markedByRegistry+string(filepath.Separator), exists)
	if len(dirs) != 1 || !sameDirectory(dirs[0], portable) {
		t.Fatalf("dirs = %#v, want only %q", dirs, portable)
	}
}

func TestInstallationResultPassesWithoutConflict(t *testing.T) {
	only := filepath.Join("root", "NodePaper")
	cases := []struct {
		name         string
		pathDirs     []string
		runningDir   string
		portableDirs []string
		setupDir     string
	}{
		{name: "nothing on PATH", runningDir: filepath.Join("build", "bin")},
		{name: "one copy", pathDirs: []string{only}, runningDir: only},
		{
			// Setup installed over the extracted folder: its uninstaller now
			// sits there, so the folder is no longer a portable installation
			// and there is no second channel to report.
			name:       "one directory taken over by Setup",
			pathDirs:   []string{only},
			runningDir: only,
			setupDir:   only + string(filepath.Separator),
		},
		{
			// A portable release and no Setup installation anywhere.
			name:         "portable release only",
			pathDirs:     []string{only},
			runningDir:   only,
			portableDirs: []string{only},
		},
		{
			// A development build run by full path is not a second
			// installation, so a single registered copy stays a pass.
			name:       "run from outside PATH",
			pathDirs:   []string{only},
			runningDir: filepath.Join("build", "bin"),
			setupDir:   only,
		},
		{
			// Two directories, but the one that wins is the running one:
			// nothing is shadowed.
			name:       "running copy wins",
			pathDirs:   []string{only, filepath.Join("root", "Older")},
			runningDir: only,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := installationResult(tc.pathDirs, tc.runningDir, tc.portableDirs, tc.setupDir)
			if check.Status != StatusPass {
				t.Fatalf("Status = %v, want StatusPass; check = %#v", check.Status, check)
			}
			if check.Name != installationCheckName {
				t.Fatalf("Name = %q, want %q", check.Name, installationCheckName)
			}
			if check.Suggestion != "" {
				t.Fatalf("a passing check must ask for nothing, got %q", check.Suggestion)
			}
			if check.Message == "" {
				t.Fatalf("Message must not be empty")
			}
			// Short and neutral: no scolding for a machine with nothing wrong.
			if strings.Contains(check.Message, "\n") {
				t.Fatalf("Message must stay on one line: %q", check.Message)
			}
		})
	}
}

func TestInstallationResultWarnsAboutShadowing(t *testing.T) {
	winner := filepath.Join("root", "Old")
	running := filepath.Join("root", "New")

	check := installationResult([]string{winner, running}, running, []string{winner, running}, "")
	if check.Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning; check = %#v", check.Status, check)
	}
	// Both directories have to be named, and which one wins has to be
	// unambiguous - the whole point is that the user cannot tell otherwise.
	if !strings.Contains(check.Message, winner) || !strings.Contains(check.Message, running) {
		t.Fatalf("Message must name both directories: %q", check.Message)
	}
	if !strings.Contains(check.Message, "comes first") {
		t.Fatalf("Message must say which directory wins: %q", check.Message)
	}
	if !strings.Contains(check.Suggestion, winner) {
		t.Fatalf("Suggestion must name the directory to act on: %q", check.Suggestion)
	}
}

func TestInstallationResultWarnsAboutTwoChannels(t *testing.T) {
	portable := filepath.Join("root", "portable")
	setup := filepath.Join("programs", "NodePaper")

	// Only the Setup directory is on PATH, so it is the one that answers. The
	// portable folder is not on PATH at all, which is still two channels.
	check := installationResult([]string{setup}, setup, []string{portable}, setup+string(filepath.Separator))
	if check.Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning; check = %#v", check.Status, check)
	}
	if !strings.Contains(check.Message, portable) || !strings.Contains(check.Message, setup) {
		t.Fatalf("Message must name both channels: %q", check.Message)
	}
	if !strings.Contains(check.Message, "comes first on PATH") {
		t.Fatalf("Message must say which channel answers to nodepaper: %q", check.Message)
	}
	if !strings.Contains(check.Suggestion, "Uninstall-NodePaper.ps1") ||
		!strings.Contains(check.Suggestion, "Windows Settings") {
		t.Fatalf("Suggestion must offer both ways out: %q", check.Suggestion)
	}
}

func TestDisplayDirectoryDropsTrailingSeparators(t *testing.T) {
	dir := filepath.Join("root", "NodePaper")
	if got := displayDirectory(dir + string(filepath.Separator)); got != dir {
		t.Fatalf("displayDirectory = %q, want %q", got, dir)
	}
	if got := displayDirectory(`"` + dir + `"`); got != dir {
		t.Fatalf("a quoted entry must print unquoted, got %q", got)
	}
	// A drive root keeps its separator: it is part of the name there.
	if got := displayDirectory(`C:\`); got != `C:\` {
		t.Fatalf("displayDirectory = %q, want the drive root unchanged", got)
	}
}

func TestInstallationResultPrintsOneSpellingPerDirectory(t *testing.T) {
	// Setup records InstallLocation with a trailing separator and a Path entry
	// usually does not; the message must not expose that difference.
	setup := filepath.Join("programs", "NodePaper")
	portable := filepath.Join("root", "portable")
	check := installationResult([]string{setup}, setup, []string{portable}, setup+string(filepath.Separator))
	if strings.Contains(check.Message, setup+string(filepath.Separator)) {
		t.Fatalf("Message shows a trailing separator: %q", check.Message)
	}
}

func TestInstallationResultReportsBothConflictsAtOnce(t *testing.T) {
	portable := filepath.Join("root", "portable")
	setup := filepath.Join("programs", "NodePaper")
	running := filepath.Join("root", "third")

	check := installationResult([]string{portable, setup, running}, running, []string{portable, running}, setup)
	if check.Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning", check.Status)
	}
	if !strings.Contains(check.Message, "comes first and answers to nodepaper") {
		t.Fatalf("Message lost the shadowing conflict: %q", check.Message)
	}
	if !strings.Contains(check.Message, "two channels are installed") {
		t.Fatalf("Message lost the two-channel conflict: %q", check.Message)
	}
	if len(strings.Split(check.Suggestion, "\n")) != 2 {
		t.Fatalf("both conflicts must carry their own next step: %q", check.Suggestion)
	}
}

func TestCheckInstallationOnThisMachine(t *testing.T) {
	checks := checkInstallation()
	if runtime.GOOS != "windows" {
		if checks != nil {
			t.Fatalf("off Windows the check must be absent, got %#v", checks)
		}
		return
	}
	if len(checks) != 1 {
		t.Fatalf("checks = %#v, want exactly one", checks)
	}
	check := checks[0]
	if check.Status != StatusPass && check.Status != StatusWarning {
		t.Fatalf("Status = %v, want pass or warning; check = %#v", check.Status, check)
	}
	if check.Name != installationCheckName {
		t.Fatalf("Name = %q, want %q", check.Name, installationCheckName)
	}
	if check.Message == "" {
		t.Fatalf("Message must not be empty")
	}
}

func TestCheckInstallationSeesAShadowingCopy(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATH shadowing between NodePaper installation channels is a Windows state")
	}

	// Two directories holding a nodepaper.exe, neither of them the test
	// binary's own directory: the first must be reported as the winner. Only
	// the process PATH is touched, and t.Setenv restores it.
	first := t.TempDir()
	second := t.TempDir()
	for _, dir := range []string{first, second} {
		if err := os.WriteFile(filepath.Join(dir, nodePaperExeName), []byte("stub"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", first+string(os.PathListSeparator)+second)

	checks := checkInstallation()
	if len(checks) != 1 {
		t.Fatalf("checks = %#v, want exactly one", checks)
	}
	if checks[0].Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning; check = %#v", checks[0].Status, checks[0])
	}
	if !strings.Contains(checks[0].Message, first) {
		t.Fatalf("Message must name the winning directory %q: %q", first, checks[0].Message)
	}
}
