package schema

import (
	"testing"
)

const fullSchema = `
version: "1.0"
types:
  user: {}
  folder:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - owner
          - parent->view
      edit:
        resolve: owner
  document:
    relations:
      owner:
        types: [user]
      viewer:
        types: [user]
      parent:
        types: [folder]
    permissions:
      view:
        union:
          - viewer
          - owner
          - parent->view
      delete:
        resolve: owner
`

func TestCompile(t *testing.T) {
	data := []byte(fullSchema)
	raw, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	cs, err := Compile(raw, data)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if cs.Hash == "" {
		t.Error("expected non-empty hash")
	}

	// Check compiled structure
	docPerm, err := cs.GetPermission("document", "view")
	if err != nil {
		t.Fatalf("GetPermission: %v", err)
	}
	if docPerm.Operation != PermOpUnion {
		t.Errorf("expected UNION, got %d", docPerm.Operation)
	}
	if len(docPerm.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(docPerm.Children))
	}

	// Verify arrow ref is parsed correctly
	var arrowFound bool
	for _, child := range docPerm.Children {
		if child.Kind() == "arrow" && child.ArrowRef == "parent" && child.ArrowPermission == "view" {
			arrowFound = true
		}
	}
	if !arrowFound {
		t.Error("expected parent->view arrow ref in document view permission")
	}
}

func TestCompileWithCondition(t *testing.T) {
	data := []byte(`
version: "1.0"
types:
  user: {}
  document:
    relations:
      owner:
        types: [user]
    permissions:
      delete:
        resolve: owner
        condition: "subject.clearance_level >= 3"
`)
	raw, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cs, err := Compile(raw, data)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	perm, err := cs.GetPermission("document", "delete")
	if err != nil {
		t.Fatalf("GetPermission: %v", err)
	}
	if perm.Condition == nil {
		t.Error("expected compiled condition")
	}
	if perm.Condition.Compiled == nil {
		t.Error("expected compiled program")
	}
}

func TestValidateSchema(t *testing.T) {
	data := []byte(fullSchema)
	raw, _ := Parse(data)
	cs, _ := Compile(raw, data)

	errs := Validate(cs)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("validation error: %v", e)
		}
	}
}
