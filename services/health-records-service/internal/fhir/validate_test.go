package fhir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDemoBundle(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "examples", "sample-fhir-data", "demo-patient-bundle.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}

	resources, err := ValidateBundleJSON(string(raw))
	if err != nil {
		t.Fatalf("validate bundle: %v", err)
	}
	if len(resources) < 5 {
		t.Fatalf("expected at least 5 resources, got %d", len(resources))
	}
}

func TestNormalizeCondition(t *testing.T) {
	raw := []byte(`{
	  "resourceType":"Condition",
	  "id":"cond-1",
	  "clinicalStatus":{"coding":[{"code":"active"}]},
	  "code":{"text":"Type 2 diabetes mellitus"}
	}`)
	norm, err := NormalizeResource(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if norm.DisplayText != "Type 2 diabetes mellitus" {
		t.Fatalf("unexpected display text: %q", norm.DisplayText)
	}
	if norm.SearchableText == "" {
		t.Fatal("expected searchable text")
	}
}

func TestValidateRejectsUnsupportedType(t *testing.T) {
	raw := []byte(`{"resourceType":"Invoice","id":"x"}`)
	_, _, err := ValidateResourceJSON(raw)
	if err == nil {
		t.Fatal("expected error for unsupported resource")
	}
}
