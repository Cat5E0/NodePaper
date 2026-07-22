package app

import (
	"context"
	"reflect"
	"testing"
)

type contractStub struct{}

func (contractStub) Init(context.Context, InitRequest) (InitResult, error) {
	return InitResult{}, nil
}

func (contractStub) Doctor(context.Context, DoctorRequest) (DoctorResult, error) {
	return DoctorResult{}, nil
}

func (contractStub) Validate(context.Context, ValidateRequest) (ValidateResult, error) {
	return ValidateResult{}, nil
}

func (contractStub) Build(context.Context, BuildRequest) (BuildResult, error) {
	return BuildResult{}, nil
}

func (contractStub) Clean(context.Context, CleanRequest) (CleanResult, error) {
	return CleanResult{}, nil
}

var _ App = contractStub{}

func TestRequestsExposeProjectDir(t *testing.T) {
	want := `D:\papers\cumcm-a`
	requests := []any{
		InitRequest{ProjectDir: want},
		DoctorRequest{ProjectDir: want},
		ValidateRequest{ProjectDir: want},
		BuildRequest{ProjectDir: want},
		CleanRequest{ProjectDir: want},
	}

	for _, request := range requests {
		value := reflect.ValueOf(request)
		if got := value.FieldByName("ProjectDir").String(); got != want {
			t.Errorf("%T.ProjectDir = %q, want %q", request, got, want)
		}
	}
}

func TestResultJSONFieldNames(t *testing.T) {
	tests := []struct {
		name   string
		result any
		fields map[string]string
	}{
		{"init", InitResult{}, map[string]string{"Success": "success", "ProjectRoot": "projectRoot", "Artifacts": "artifacts", "Diagnostics": "diagnostics"}},
		{"doctor", DoctorResult{}, map[string]string{"Success": "success", "ProjectRoot": "projectRoot", "Diagnostics": "diagnostics"}},
		{"validate", ValidateResult{}, map[string]string{"Success": "success", "ProjectRoot": "projectRoot", "Diagnostics": "diagnostics"}},
		{"build", BuildResult{}, map[string]string{"Success": "success", "BuildID": "buildId", "ProjectRoot": "projectRoot", "Artifacts": "artifacts", "Diagnostics": "diagnostics"}},
		{"clean", CleanResult{}, map[string]string{"Success": "success", "ProjectRoot": "projectRoot", "Artifacts": "artifacts", "Diagnostics": "diagnostics"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			typeOf := reflect.TypeOf(test.result)
			for fieldName, wantTag := range test.fields {
				field, ok := typeOf.FieldByName(fieldName)
				if !ok {
					t.Fatalf("%T has no field %s", test.result, fieldName)
				}
				if got := field.Tag.Get("json"); got != wantTag {
					t.Errorf("%T.%s json tag = %q, want %q", test.result, fieldName, got, wantTag)
				}
			}
		})
	}
}
