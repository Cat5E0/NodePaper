package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// installationCheckName is the fixed Name of the check below.
const installationCheckName = "NodePaper installation"

// nodePaperExeName is the command Windows resolves on PATH.
const nodePaperExeName = "nodepaper.exe"

// How the two installation channels are recognised.
//
// Setup records its directory as InstallLocation under its own AppId in HKCU
// (per-user: it needs no administrator rights) and writes its uninstaller into
// that directory.
//
// The portable ZIP channel records nothing anywhere. A directory holding a
// nodepaper.exe and no unins000.exe is an extracted release, and that is the
// whole of its identity: Install-NodePaper.ps1 used to write the one directory
// it had registered to HKCU\Software\NodePaper\PortablePath, a single global
// value that could describe only one folder, went stale when Setup took the
// folder over or the folder was emptied, and did not travel with a folder copied
// to another drive. Reading PATH back instead cannot go stale, and it reports
// the folders that actually answer to `nodepaper`.
const (
	setupUninstallKey    = `Software\Microsoft\Windows\CurrentVersion\Uninstall\{6E1B5C6A-6C2F-4D4B-9A62-2C7E60C0A5F1}_is1`
	setupLocationValue   = "InstallLocation"
	setupUninstallerName = "unins000.exe"
)

// checkInstallation reports whether more than one NodePaper answers to
// `nodepaper` on this machine. Two states are detectable and both leave the
// user reading the output of a copy they did not mean to run:
//
//   - Several directories on PATH hold nodepaper.exe. Windows searches PATH
//     left to right, so the first one wins and any later one is shadowed -
//     which is what `where nodepaper` shows and what registration cannot fix,
//     because a Path entry is appended, never promoted.
//   - Both channels are installed at different directories: a Setup
//     installation, and a portable ZIP release on PATH. Each is internally
//     consistent, so neither channel notices the other.
//
// This lives in doctor rather than in the installers on purpose: it is a
// property of the machine, not of one installation run, and it can change
// after any install (a hand-edited Path, a second ZIP extracted later). The
// same reasoning is why brew doctor, rustup, nvm debug and flutter doctor all
// carry their shadowing checks in a command that can be re-run, and brew's
// rule is followed here too: when nothing conflicts, the check says so in one
// short line and asks for nothing.
//
// Returns no check at all off Windows: PATH shadowing between installation
// channels that only exist on Windows cannot be diagnosed there, and a
// permanently skipped check would just add a warning to every run.
func checkInstallation() []Check {
	if runtime.GOOS != "windows" {
		return nil
	}

	runningDir := ""
	if executable, err := os.Executable(); err == nil {
		runningDir = filepath.Dir(executable)
	}

	// The process PATH is what a command lookup uses, and Windows has already
	// expanded any %VARIABLE% entry in it by the time it reaches this process.
	pathDirs := nodePaperDirsOnPath(os.Getenv("PATH"), fileExists)

	setupDir, _ := readRegistryString(setupUninstallKey, setupLocationValue)
	portableDirs := portableDirsAmong(pathDirs, setupDir, fileExists)

	return []Check{installationResult(pathDirs, runningDir, portableDirs, setupDir)}
}

// portableDirsAmong keeps the directories that hold an extracted ZIP release,
// in the order they were given. Setup's directory is excluded by either of its
// marks - its uninstaller sitting in the directory, or its uninstall entry
// naming the directory - which is the same pair of marks Install-NodePaper.ps1
// and Uninstall-NodePaper.ps1 refuse to act on.
func portableDirsAmong(dirs []string, setupDir string, exists func(string) bool) []string {
	var portable []string
	for _, dir := range dirs {
		if sameDirectory(dir, setupDir) {
			continue
		}
		if exists(filepath.Join(dir, setupUninstallerName)) {
			continue
		}
		portable = append(portable, dir)
	}
	return portable
}

// installationResult renders the collected state into a Check. It is kept free
// of PATH and registry access so every combination can be exercised directly
// in tests.
func installationResult(pathDirs []string, runningDir string, portableDirs []string, setupDir string) Check {
	var conflicts, suggestions []string

	// Shadowing: PATH holds more than one NodePaper and the one that wins is
	// not the executable this doctor run came from. A single directory cannot
	// shadow anything, and a run started by full path from outside PATH (a
	// development build, an extracted ZIP nobody registered) is not evidence
	// of a second installation on its own.
	if len(pathDirs) > 1 && runningDir != "" && !sameDirectory(pathDirs[0], runningDir) {
		conflicts = append(conflicts, fmt.Sprintf(
			"%d directories on PATH hold %s; %s comes first and answers to nodepaper, ahead of this executable in %s",
			len(pathDirs), nodePaperExeName, displayDirectory(pathDirs[0]), displayDirectory(runningDir)))
		suggestions = append(suggestions, fmt.Sprintf(
			"Keep one NodePaper on Path: remove %s from Path or uninstall that copy, then open a new terminal.",
			displayDirectory(pathDirs[0])))
	}

	// Two channels installed at two different directories: a portable release
	// on PATH plus a Setup installation. One directory claimed by both is not a
	// conflict and cannot arise any more: installing over a portable folder puts
	// Setup's uninstaller in it, which stops that folder from counting as
	// portable at all.
	//
	// The first portable directory that is not Setup's is the one named, because
	// it is the one the suggestion acts on; PATH order decides which that is.
	if setupDir != "" && len(portableDirs) > 0 {
		portableDir := portableDirs[0]
		conflict := fmt.Sprintf(
			"two channels are installed: a portable ZIP release in %s and a Setup installation in %s",
			displayDirectory(portableDir), displayDirectory(setupDir))
		if winner := firstDirOnPath(pathDirs, portableDir, setupDir); winner != "" {
			conflict += fmt.Sprintf("; %s comes first on PATH", displayDirectory(winner))
		}
		conflicts = append(conflicts, conflict)
		suggestions = append(suggestions, fmt.Sprintf(
			"Keep one channel: run Uninstall-NodePaper.ps1 in %s, or uninstall NodePaper from Windows Settings.",
			displayDirectory(portableDir)))
	}

	if len(conflicts) > 0 {
		return Check{
			Name:       installationCheckName,
			Status:     StatusWarning,
			Message:    strings.Join(conflicts, "; "),
			Suggestion: strings.Join(suggestions, "\n"),
		}
	}

	switch {
	case len(pathDirs) == 0:
		return Check{Name: installationCheckName, Status: StatusPass, Message: "no NodePaper directory on PATH"}
	case len(pathDirs) == 1:
		return Check{Name: installationCheckName, Status: StatusPass, Message: displayDirectory(pathDirs[0])}
	default:
		return Check{
			Name:    installationCheckName,
			Status:  StatusPass,
			Message: fmt.Sprintf("%s (first of %d NodePaper directories on PATH)", displayDirectory(pathDirs[0]), len(pathDirs)),
		}
	}
}

// nodePaperDirsOnPath returns the directories of pathValue that hold
// nodepaper.exe, in PATH order and without repeats. That order is the
// resolution order: Windows and where.exe both search PATH left to right and
// stop at the first hit.
func nodePaperDirsOnPath(pathValue string, exists func(string) bool) []string {
	var dirs []string
	seen := make(map[string]struct{})
	for _, entry := range strings.Split(pathValue, string(os.PathListSeparator)) {
		// A user PATH routinely holds entries that cannot be probed: a quoted
		// entry, a stale directory, a disconnected drive. Each is skipped in
		// turn rather than allowed to end the scan.
		entry = strings.TrimSpace(strings.Trim(strings.TrimSpace(entry), `"`))
		if entry == "" {
			continue
		}
		key := normalizeDirectory(entry)
		if key == "" {
			continue
		}
		if _, repeated := seen[key]; repeated {
			continue
		}
		seen[key] = struct{}{}
		if exists(filepath.Join(entry, nodePaperExeName)) {
			dirs = append(dirs, entry)
		}
	}
	return dirs
}

// firstDirOnPath returns whichever of the candidates comes first in pathDirs,
// or "" when none of them is on PATH at all.
func firstDirOnPath(pathDirs []string, candidates ...string) string {
	for _, dir := range pathDirs {
		for _, candidate := range candidates {
			if sameDirectory(dir, candidate) {
				return dir
			}
		}
	}
	return ""
}

// sameDirectory compares two directories the way the installers do: quotes
// stripped, trailing separators dropped, case folded. Setup writes
// InstallLocation with a trailing backslash while a Path entry usually has
// none, so the same directory would otherwise look like two.
func sameDirectory(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return normalizeDirectory(left) == normalizeDirectory(right)
}

func normalizeDirectory(dir string) string {
	dir = strings.TrimSpace(strings.Trim(strings.TrimSpace(dir), `"`))
	if dir == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(dir))
}

// displayDirectory spells a directory the way the message should read. The
// spellings in hand differ - Setup's InstallLocation ends in a backslash, a
// Path entry may or may not, and either may be quoted - and printing them
// verbatim made one machine's two installations look like three different
// spellings. Only the presentation is affected; comparison still goes through
// normalizeDirectory.
func displayDirectory(dir string) string {
	dir = strings.TrimSpace(strings.Trim(strings.TrimSpace(dir), `"`))
	for len(dir) > 0 && os.IsPathSeparator(dir[len(dir)-1]) {
		trimmed := dir[:len(dir)-1]
		// "C:\" is a directory whose separator is part of its name.
		if strings.HasSuffix(trimmed, ":") {
			break
		}
		dir = trimmed
	}
	return dir
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
