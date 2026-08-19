package validate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type fixtureManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Fixtures      []fixtureContract `json:"fixtures"`
}

type fixtureContract struct {
	Path                    string   `json:"path"`
	Command                 string   `json:"command"`
	ExpectedSuccess         bool     `json:"expectedSuccess"`
	ExpectedExitCode        int      `json:"expectedExitCode"`
	ExpectedDiagnosticCodes []string `json:"expectedDiagnosticCodes"`
}

func TestValidateFixtureContracts(t *testing.T) {
	repoRoot := repositoryRoot(t)
	assetRoot := filepath.Join(repoRoot, "tests")
	manifest := readFixtureManifest(t, filepath.Join(assetRoot, "fixture-manifest.json"))
	if manifest.SchemaVersion != 2 {
		t.Fatalf("fixture manifest schemaVersion = %d, want 2", manifest.SchemaVersion)
	}

	for _, contract := range manifest.Fixtures {
		contract := contract
		if contract.Command != "validate" {
			continue
		}
		t.Run(contract.Path, func(t *testing.T) {
			source := filepath.Join(assetRoot, "fixtures", contract.Path)
			before := snapshotFiles(t, source)
			projectDir := filepath.Join(t.TempDir(), "project")
			copyTree(t, source, projectDir)

			result := Run(context.Background(), projectDir)
			if result.Success != contract.ExpectedSuccess {
				t.Fatalf("Run().Success = %v, want %v; diagnostics = %#v", result.Success, contract.ExpectedSuccess, result.Diagnostics)
			}

			// Fixture contracts describe what is wrong with the Project, so
			// they must hold on any machine. Diagnostics sourced from "font"
			// describe the host instead (NP2403 fires wherever the Chinese
			// supplemental fonts are absent, which is every CI runner), and
			// pinning them either way would make this test pass on one class
			// of machine and fail on the other.
			gotCodes := make([]string, 0, len(result.Diagnostics))
			for _, diag := range result.Diagnostics {
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
				t.Fatalf("diagnostic codes = %v, want %v; diagnostics = %#v", gotCodes, wantCodes, result.Diagnostics)
			}

			after := snapshotFiles(t, source)
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("source fixture was modified: before=%v after=%v", before, after)
			}
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func readFixtureManifest(t *testing.T, path string) fixtureManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse fixture manifest: %v", err)
	}
	return manifest
}

func copyTree(t *testing.T, source, destination string) {
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
		t.Fatalf("copy fixture tree: %v", err)
	}
}

func snapshotFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		snapshot[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot fixture: %v", err)
	}
	return snapshot
}
