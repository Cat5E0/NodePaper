// Package fonts answers one question for every part of NodePaper that has to
// ask it: are the optional Chinese supplemental fonts installed on this
// machine? Both `nodepaper validate` (which warns the author before a build)
// and `nodepaper doctor` (whose Chinese probe has to compile the same document
// the build will compile) depend on the answer, and they must never disagree -
// a doctor that reports a broken environment while build produces a PDF is
// worse than no doctor at all.
package fonts

import (
	"os"
	"path/filepath"
)

// Supplemental describes one font from the optional Windows "Chinese
// (Simplified) Supplemental Fonts" feature: the family name ctex asks for, the
// file the feature installs, and the style NodePaper synthesises from SimSun
// when it is absent.
type Supplemental struct {
	Name string
	File string
	Role string
}

// Supplementals are shipped by Windows as the optional "Chinese (Simplified)
// Supplemental Fonts" feature, so a machine that never added Chinese can be
// missing them while everything else works.
var Supplementals = []Supplemental{
	{Name: "SimHei", File: "simhei.ttf", Role: "bold"},
	{Name: "KaiTi", File: "simkai.ttf", Role: "italic"},
}

// Lookup is the outcome of looking for a single font file. The third state
// matters: a font directory that cannot be read proves nothing, and treating
// that as "absent" would produce warnings users learn to ignore.
type Lookup int

const (
	Present Lookup = iota
	Missing
	Unknown
)

// Availability is what a probe of Supplementals could establish.
type Availability struct {
	// Missing lists the fonts proven absent, in Supplementals order.
	Missing []Supplemental
	// Undetermined is true when at least one font could not be classified
	// because a font directory could not be read.
	Undetermined bool
}

// AllPresent reports whether every supplemental font was positively found.
// It is deliberately false when the probe was undetermined, so callers that
// have to pick a code path pick the one that also works without the fonts.
func (a Availability) AllPresent() bool {
	return !a.Undetermined && len(a.Missing) == 0
}

// Names returns the family names of the missing fonts.
func (a Availability) Names() []string {
	names := make([]string, 0, len(a.Missing))
	for _, font := range a.Missing {
		names = append(names, font.Name)
	}
	return names
}

// ProbeSupplemental classifies every font in Supplementals. It looks only at
// the two directories Windows installs fonts into, mirroring the probe in
// Convert-CumcmProjectToLatex.ps1 that decides the real build's preamble.
func ProbeSupplemental() Availability {
	var availability Availability
	for _, font := range Supplementals {
		switch InstalledFile(font.File) {
		case Missing:
			availability.Missing = append(availability.Missing, font)
		case Unknown:
			availability.Undetermined = true
		}
	}
	return availability
}

// InstalledFile reports whether a font file of this name is installed, for the
// machine's fonts and the per-user fonts alike.
func InstalledFile(name string) Lookup {
	dirs := make([]string, 0, 2)
	if windir := os.Getenv("WINDIR"); windir != "" {
		dirs = append(dirs, filepath.Join(windir, "Fonts"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
	}
	if len(dirs) == 0 {
		return Unknown
	}
	for _, dir := range dirs {
		switch _, err := os.Stat(filepath.Join(dir, name)); {
		case err == nil:
			return Present
		case os.IsNotExist(err):
			continue
		default:
			return Unknown
		}
	}
	return Missing
}
