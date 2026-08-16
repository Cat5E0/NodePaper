package latexlog

import "testing"

// Real fontspec output wraps at about 80 columns, so the family name, the
// "with options" tail and the bracket contents each land on their own line.
const wrappedLog = `Package fontspec Info:
(fontspec)             Font family 'SimSun(0)' created for font 'SimSun' with
(fontspec)             options
(fontspec)             [Script={CJK},BoldFont={SimHei},ItalicFont={KaiTi}].
(fontspec)
(fontspec)              This font family consists of the following NFSS
(fontspec)              series/shapes:
LaTeX Font Info:    Checking defaults for OML/cmm/m/it on input line 219.
Package fontspec Info:
(fontspec)             Font family 'SimHei(0)' created for font 'SimHei' with
(fontspec)             options [Script={CJK}].
`

func TestFontsJoinsWrappedLines(t *testing.T) {
	families := Fonts([]byte(wrappedLog))
	if len(families) != 2 {
		t.Fatalf("families = %#v, want 2", families)
	}
	first := families[0]
	if first.Family != "SimSun(0)" || first.Font != "SimSun" {
		t.Fatalf("first = %#v", first)
	}
	// The roles only survive if the bracket line was joined to the line above.
	if first.Options != "Script={CJK},BoldFont={SimHei},ItalicFont={KaiTi}" {
		t.Fatalf("first.Options = %q; wrapped bracket line was not joined", first.Options)
	}
	if first.Line != 2 {
		t.Fatalf("first.Line = %d, want 2", first.Line)
	}
	if families[1].Font != "SimHei" || families[1].Options != "Script={CJK}" {
		t.Fatalf("second = %#v", families[1])
	}
}

// A fallback that binds every family to one font is only visible as repeated
// entries, so deduplicating by font name would erase the thing worth seeing.
func TestFontsKeepsRepeatedFamiliesOfOneFont(t *testing.T) {
	log := `Package fontspec Info:
(fontspec)             Font family 'SimSun(0)' created for font 'SimSun' with
(fontspec)             options [Script={CJK},AutoFakeBold=true].
Package fontspec Info:
(fontspec)             Font family 'SimSun(1)' created for font 'SimSun' with
(fontspec)             options [Script={CJK},AutoFakeBold=true].
`
	families := Fonts([]byte(log))
	if len(families) != 2 {
		t.Fatalf("families = %#v, want both SimSun entries", families)
	}
	if families[0].Family == families[1].Family {
		t.Fatalf("families collapsed: %#v", families)
	}
}

func TestFontsWithoutOptions(t *testing.T) {
	log := "Package fontspec Info: \n(fontspec)             Font family 'Fandol(0)' created for font 'FandolSong'\n"
	families := Fonts([]byte(log))
	if len(families) != 1 || families[0].Font != "FandolSong" {
		t.Fatalf("families = %#v", families)
	}
	if families[0].Options != "" {
		t.Fatalf("Options = %q, want empty", families[0].Options)
	}
}

// Profiles without CJK produce no fontspec records at all; that is not an error.
func TestFontsOnLogWithoutFontspec(t *testing.T) {
	if families := Fonts([]byte("This is TeX, Version 3.141592653\nOutput written on paper.pdf.\n")); families != nil {
		t.Fatalf("families = %#v, want nil", families)
	}
	if families := Fonts(nil); families != nil {
		t.Fatalf("families = %#v, want nil", families)
	}
}

func TestEngineFromLogHeader(t *testing.T) {
	log := "This is XeTeX, Version 3.141592653-2.6-0.999997 (TeX Live 2025) (preloaded format=xelatex 2025.4.17)  13 AUG 2026 17:45\nentering extended mode\n"
	got := Engine([]byte(log))
	want := "XeTeX, Version 3.141592653-2.6-0.999997 (TeX Live 2025)"
	if got != want {
		t.Fatalf("Engine = %q, want %q", got, want)
	}
}

// A log that names no engine is not an error; the field is simply empty.
func TestEngineAbsent(t *testing.T) {
	if got := Engine([]byte("entering extended mode\nOutput written on paper.pdf.\n")); got != "" {
		t.Fatalf("Engine = %q, want empty", got)
	}
	if got := Engine(nil); got != "" {
		t.Fatalf("Engine = %q, want empty", got)
	}
}
