package cooldown

import (
	"testing"
	"time"
)

func TestEligible_Disabled(t *testing.T) {
	now := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	if !Eligible("", 0, now) {
		t.Fatalf("expected eligible when minDays=0")
	}
}

func TestEligible_EmptyOrInvalidTime(t *testing.T) {
	now := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	if Eligible("", 30, now) {
		t.Fatalf("expected ineligible for empty time when cooldown enabled")
	}
	if Eligible("not-a-time", 30, now) {
		t.Fatalf("expected ineligible for invalid time when cooldown enabled")
	}
}

func TestEligible_AgeThreshold(t *testing.T) {
	now := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	old := now.Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	recent := now.Add(-29 * 24 * time.Hour).Format(time.RFC3339)

	if !Eligible(old, 30, now) {
		t.Fatalf("expected eligible when older than threshold")
	}
	if Eligible(recent, 30, now) {
		t.Fatalf("expected ineligible when newer than threshold")
	}
}
