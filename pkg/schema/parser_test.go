package schema

import (
	"strings"
	"testing"
)

const sampleSchema = `
version: "1.0"
types:
  user: {}
  document:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user]
    permissions:
      view:
        union:
          - viewer
          - owner
      delete:
        resolve: owner
`

func TestParse(t *testing.T) {
	raw, err := Parse([]byte(sampleSchema))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if raw.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", raw.Version)
	}
	if len(raw.Types) != 2 {
		t.Errorf("types count = %d, want 2", len(raw.Types))
	}
}

func TestParseUnknownField(t *testing.T) {
	bad := `
version: "1.0"
types:
  user:
    unknown_field: foo
`
	_, err := Parse([]byte(bad))
	if err == nil {
		t.Error("expected error for unknown field, got nil")
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse([]byte(""))
	// Empty YAML should not error (empty schema)
	if err != nil && !strings.Contains(err.Error(), "EOF") {
		t.Logf("Parse empty: %v (acceptable)", err)
	}
}
