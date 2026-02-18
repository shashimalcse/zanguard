package model

import (
	"testing"
)

func TestTupleKey(t *testing.T) {
	tests := []struct {
		name     string
		tuple    RelationTuple
		expected string
	}{
		{
			name: "simple tuple",
			tuple: RelationTuple{
				ObjectType:  "document",
				ObjectID:    "readme",
				Relation:    "viewer",
				SubjectType: "user",
				SubjectID:   "thilina",
			},
			expected: "document:readme#viewer@user:thilina",
		},
		{
			name: "userset tuple",
			tuple: RelationTuple{
				ObjectType:      "document",
				ObjectID:        "readme",
				Relation:        "viewer",
				SubjectType:     "group",
				SubjectID:       "engineering",
				SubjectRelation: "member",
			},
			expected: "document:readme#viewer@group:engineering#member",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.tuple.TupleKey()
			if got != tc.expected {
				t.Errorf("TupleKey() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestTupleKeyNoCollisions(t *testing.T) {
	tuples := []RelationTuple{
		{ObjectType: "doc", ObjectID: "a", Relation: "viewer", SubjectType: "user", SubjectID: "x"},
		{ObjectType: "doc", ObjectID: "a#viewer@user", Relation: "other", SubjectType: "y", SubjectID: "z"},
	}
	keys := make(map[string]bool)
	for _, tup := range tuples {
		k := tup.TupleKey()
		if keys[k] {
			t.Errorf("collision detected for key %q", k)
		}
		keys[k] = true
	}
}
