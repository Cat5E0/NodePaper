package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type cliFixtureManifest struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Fixtures      []cliFixtureContract `json:"fixtures"`
}

type cliFixtureContract struct {
	Path                    string   `json:"path"`
	Command                 string   `json:"command"`
	ExpectedSuccess         bool     `json:"expectedSuccess"`
	ExpectedExitCode        int      `json:"expectedExitCode"`
	ExpectedDiagnosticCodes []string `json:"expectedDiagnosticCodes"`
}

type validateEnvelope struct {
	SchemaVersion int `json:"schemaVersion"`
	Success       bool
	ProjectRoot   string `json:"projectRoot"`
	Diagnostics   []struct {
		Code string `json:"code"`
		// Source separates what is wrong with the Project from what is
		// merely true of this machine. See the filter below.
		Source string `json:"source"`
	} `json:"diagnostics"`
}

func TestFixtureJSONContracts(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODEPAPER_PROFILE_DIR", filepath.Join(repoRoot, "profiles", "cumcm"))
	assetRoot := filepath.Join(repoRoot, "tests")
	manifestData, err := os.ReadFile(filepath.Join(assetRoot, "fixture-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest cliFixtureManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 2 {
		t.Fatalf("manifest schemaVersion = %d, want 2", manifest.SchemaVersion)
	}

	for _, contract := range manifest.Fixtures {
		contract := contract
		if contract.Command != "validate" && contract.Command != "build" {
			continue
		}
		t.Run(contract.Command+"/"+contract.Path, func(t *testing.T) {
			projectDir := filepath.Join(t.TempDir(), "project")
			copyCLIFixture(t, filepath.Join(assetRoot, "fixtures", contract.Path), projectDir)

			var stdout, stderr bytes.Buffer
			exitCode := run([]string{contract.Command, projectDir, "--format", "json"}, &stdout, &stderr)
			if exitCode != contract.ExpectedExitCode {
				t.Fatalf("exit code = %d, want %d; stdout=%s stderr=%s", exitCode, contract.ExpectedExitCode, stdout.String(), stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty for rendered result", stderr.String())
			}

			var envelope validateEnvelope
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatalf("stdout is not one JSON object: %v\n%s", err, stdout.String())
			}
			if envelope.SchemaVersion != 1 {
				t.Fatalf("schemaVersion = %d, want 1", envelope.SchemaVersion)
			}
			if envelope.Success != contract.ExpectedSuccess {
				t.Fatalf("success = %v, want %v", envelope.Success, contract.ExpectedSuccess)
			}
			if !sameCLIPath(envelope.ProjectRoot, projectDir) {
				t.Fatalf("projectRoot = %q, want %q", envelope.ProjectRoot, projectDir)
			}

			// Fixture contracts describe the Project, so they must hold on
			// any machine. Diagnostics sourced from "font" describe the host
			// instead (NP2403 fires wherever the Chinese supplemental fonts
			// are absent, which is every CI runner), and pinning them either
			// way would make this test pass on one class of machine and fail
			// on the other. Mirrors the same filter in
			// internal/validate/fixture_test.go.
			gotCodes := make([]string, 0, len(envelope.Diagnostics))
			for _, diag := range envelope.Diagnostics {
				if diag.Source == "font" {
					continue
				}
				gotCodes = append(gotCodes, diag.Code)
			}
			sort.Strings(gotCodes)
			wantCodes := make([]string, len(contract.ExpectedDiagnosticCodes))
			copy(wantCodes, contract.ExpectedDiagnosticCodes)
			sort.Strings(wantCodes)
			if !reflect.DeepEqual(gotCodes, wantCodes) {
				t.Fatalf("diagnostic codes = %v, want %v; stdout=%s", gotCodes, wantCodes, stdout.String())
			}
		})
	}
}

func copyCLIFixture(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
}

func sameCLIPath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b) || bytes.EqualFold([]byte(filepath.Clean(a)), []byte(filepath.Clean(b)))
}
