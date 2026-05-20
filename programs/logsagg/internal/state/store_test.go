package state

import (
	"testing"
	"time"
)

func TestAdvanceRequiresMonotonicID(t *testing.T) {
	s := Progress{LastID: 100}
	err := s.Advance(99, time.Unix(1715659200, 0).UTC(), time.Unix(1715659260, 0).UTC(), false)
	if err == nil {
		t.Fatalf("expected monotonic id error")
	}
}

func TestAdvanceUpdatesTimestamps(t *testing.T) {
	now := time.Unix(1715659260, 0).UTC()
	s := Progress{}
	err := s.Advance(101, time.Unix(1715659200, 0).UTC(), now, false)
	if err != nil {
		t.Fatalf("Advance returned error: %v", err)
	}
	if s.LastID != 101 {
		t.Fatalf("expected LastID 101, got %d", s.LastID)
	}
	if !s.LastIncrementalRunAt.Equal(now) {
		t.Fatalf("expected incremental timestamp to be updated, got %s", s.LastIncrementalRunAt)
	}
	if !s.UpdatedAt.Equal(now) {
		t.Fatalf("expected UpdatedAt to be updated, got %s", s.UpdatedAt)
	}
}
