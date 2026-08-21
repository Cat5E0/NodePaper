package fonts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// pointFontDirsAt redirects the probe at a directory the test controls, so the
// machine running the suite cannot decide the outcome.
func pointFontDirsAt(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("WINDIR", dir)
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "local"))
}

func writeFontFile(t *testing.T, dir, name string) {
	t.Helper()
	fontsDir := filepath.Join(dir, "Fonts")
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fontsDir, name), []byte("stub"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func requireWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("the supplemental font probe only applies to Windows")
	}
}

func TestInstalledFileClassifies(t *testing.T) {
	requireWindows(t)
	dir := t.TempDir()
	writeFontFile(t, dir, "simhei.ttf")
	pointFontDirsAt(t, dir)

	if got := InstalledFile("simhei.ttf"); got != Present {
		t.Fatalf("InstalledFile(present) = %v, want Present", got)
	}
	if got := InstalledFile("simkai.ttf"); got != Missing {
		t.Fatalf("InstalledFile(absent) = %v, want Missing", got)
	}

	t.Setenv("WINDIR", "")
	t.Setenv("LOCALAPPDATA", "")
	if got := InstalledFile("simhei.ttf"); got != Unknown {
		t.Fatalf("InstalledFile(no dirs) = %v, want Unknown", got)
	}
}

func TestProbeSupplementalReportsWhatIsMissing(t *testing.T) {
	requireWindows(t)

	t.Run("both present", func(t *testing.T) {
		dir := t.TempDir()
		writeFontFile(t, dir, "simhei.ttf")
		writeFontFile(t, dir, "simkai.ttf")
		pointFontDirsAt(t, dir)

		availability := ProbeSupplemental()
		if !availability.AllPresent() {
			t.Fatalf("AllPresent() = false, want true; availability = %#v", availability)
		}
		if len(availability.Missing) != 0 {
			t.Fatalf("Missing = %v, want none", availability.Names())
		}
	})

	t.Run("SimHei absent", func(t *testing.T) {
		dir := t.TempDir()
		writeFontFile(t, dir, "simkai.ttf")
		pointFontDirsAt(t, dir)

		availability := ProbeSupplemental()
		if availability.AllPresent() {
			t.Fatal("AllPresent() = true with SimHei absent")
		}
		if got := strings.Join(availability.Names(), ","); got != "SimHei" {
			t.Fatalf("Names() = %q, want SimHei", got)
		}
	})

	t.Run("KaiTi absent", func(t *testing.T) {
		dir := t.TempDir()
		writeFontFile(t, dir, "simhei.ttf")
		pointFontDirsAt(t, dir)

		availability := ProbeSupplemental()
		if availability.AllPresent() {
			t.Fatal("AllPresent() = true with KaiTi absent")
		}
		if got := strings.Join(availability.Names(), ","); got != "KaiTi" {
			t.Fatalf("Names() = %q, want KaiTi", got)
		}
	})

	t.Run("both absent", func(t *testing.T) {
		pointFontDirsAt(t, t.TempDir())

		availability := ProbeSupplemental()
		if got := strings.Join(availability.Names(), ","); got != "SimHei,KaiTi" {
			t.Fatalf("Names() = %q, want SimHei,KaiTi", got)
		}
		if availability.Undetermined {
			t.Fatal("Undetermined = true although both directories were readable")
		}
	})

	// No font directory at all is not evidence of absence, and AllPresent must
	// stay false so callers pick the path that also works without the fonts.
	t.Run("undetermined", func(t *testing.T) {
		t.Setenv("WINDIR", "")
		t.Setenv("LOCALAPPDATA", "")

		availability := ProbeSupplemental()
		if !availability.Undetermined {
			t.Fatal("Undetermined = false without any font directory")
		}
		if availability.AllPresent() {
			t.Fatal("AllPresent() = true although nothing could be probed")
		}
	})
}
