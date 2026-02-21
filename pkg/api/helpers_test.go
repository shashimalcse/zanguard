package api

import (
	"testing"
	"time"
)

func TestTupleExpiryFromRequestTTL(t *testing.T) {
	now := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)
	ttl := int64(300)

	expiresAt, err := tupleExpiryFromRequest(TupleRequest{TTLSeconds: &ttl}, now)
	if err != nil {
		t.Fatalf("tupleExpiryFromRequest returned error: %v", err)
	}
	if expiresAt == nil {
		t.Fatal("expected expiresAt to be set")
	}
	if got, want := *expiresAt, now.Add(300*time.Second); !got.Equal(want) {
		t.Fatalf("expiresAt mismatch: got %s want %s", got, want)
	}
}

func TestTupleExpiryFromRequestRejectsInvalidInputs(t *testing.T) {
	now := time.Now().UTC()
	ttl := int64(10)

	_, err := tupleExpiryFromRequest(TupleRequest{
		TTLSeconds: &ttl,
		ExpiresAt:  "2026-02-21T10:00:00Z",
	}, now)
	if err == nil {
		t.Fatal("expected error for mutually exclusive ttl_seconds and expires_at")
	}

	badTTL := int64(0)
	_, err = tupleExpiryFromRequest(TupleRequest{TTLSeconds: &badTTL}, now)
	if err == nil {
		t.Fatal("expected error for ttl_seconds <= 0")
	}

	_, err = tupleExpiryFromRequest(TupleRequest{ExpiresAt: "not-a-time"}, now)
	if err == nil {
		t.Fatal("expected error for invalid expires_at")
	}
}
