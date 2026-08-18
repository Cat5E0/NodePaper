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

// Registry locations of the two installation channels. Setup records its
// directory as InstallLocation under its own AppId; Install-NodePaper.ps1
// records the extracted folder it registered as PortablePath. Both are
// per-user (HKCU) because neither channel needs administrator rights.
const (
	setupUninstallKey  = `Software\Microsoft\Windows\CurrentVersion\Uninstall\{6E1B5C6A-6C2F-4D4B-9A62-2C7E60C0A5F1}_is1`
	setupLocationValue = "InstallLocation"
	portableKey        = `Software\NodePaper`
	portableValue      = "PortablePath"
)

// checkInstallation reports whether more than one NodePaper answers to
// `nodepaper` on this machine. Two states are detectable and both leave the
// user reading the output of a copy they did not mean to run:
//
//   - Several directories on PATH hold nodepaper.exe. Windows searches PATH
//     left to right, so the first one wins and any later one is shadowed -
//     which is what `where nodepaper` shows and what registration cannot fix,
//     because a Path entry is appended, never promoted.
//   - Both channels are registered at different directories: a Setup
//     installation and a portable ZIP registration. Each is internally
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
	portableDir, _ := readRegistryString(portableKey, portableValue)

	return []Check{installationResult(pathDirs, runningDir, portableDir, setupDir)}
}

// installationResult renders the collected state into a Check. It is kept free
// of PATH and registry access so every combination can be exercised directly
// in tests.
func installationResult(pathDirs []string, runningDir, portableDir, setupDir string) Check {
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

	// Two channels claiming two different directories. The same directory in
	// both is not a conflict: that is the moment a Setup installation takes a
	// portable registration over, and Setup drops the registration itself.
	if portableDir != "" && setupDir != "" && !sameDirectory(portableDir, setupDir) {
		conflict := fmt.Sprintf(
			"two channels are installed: a portable ZIP registration in %s and a Setup installation in %s",
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
// InstallLocation with a trailing backslash and Install-NodePaper.ps1 writes
// PortablePath without one, so the two channels would otherwise look like
// different directories whenever they name the same one.
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
// two channels record their directory differently - Setup's InstallLocation
// ends in a backslash, PortablePath does not, and a Path entry may do either -
// and printing them verbatim made one machine's two installations look like
// three different spellings. Only the presentation is affected; comparison
// still goes through normalizeDirectory.
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
