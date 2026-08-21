package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDoctorExplicitInvalidProjectFailsStrictly(t *testing.T) {
	application := &appImpl{}
	result, err := application.Doctor(context.Background(), DoctorRequest{ProjectDir: filepath.Join(t.TempDir(), "missing")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "NP1001" {
		t.Fatalf("result = %#v, want NP1001", result)
	}
}
