package format

import (
	"testing"
	"time"

	"github.com/pragmaticivan/go-check-updates/internal/scanner"
)

func TestParseFlag(t *testing.T) {
	opts, err := ParseFlag("group,time")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !opts.Group || !opts.Time || opts.Lines {
		t.Fatalf("unexpected opts: %+v", opts)
	}

	_, err = ParseFlag("nope")
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}

func TestPublishTime(t *testing.T) {
	now := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	tm := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	got := PublishTime(tm, now)
	if got != "2026-01-10 (7d ago)" {
		t.Fatalf("unexpected publish time: %q", got)
	}
}

func TestParseRFC3339ish(t *testing.T) {
	if _, ok := ParseRFC3339ish("2026-01-17T00:00:00.123456789Z"); !ok {
		t.Fatalf("expected RFC3339Nano to parse")
	}
	if _, ok := ParseRFC3339ish("not-a-time"); ok {
		t.Fatalf("expected invalid time to fail")
	}
}

func TestGroupLabelAndSortKey(t *testing.T) {
	mMajor := scanner.Module{Version: "v1.0.0", Update: &scanner.Module{Version: "v2.0.0"}}
	mMinor := scanner.Module{Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}}
	mPatch := scanner.Module{Version: "v1.0.0", Update: &scanner.Module{Version: "v1.0.1"}}
	mV0Minor := scanner.Module{Version: "v0.1.0", Update: &scanner.Module{Version: "v0.2.0"}}

	if GroupLabel(mMajor) != "Major" || GroupSortKey(mMajor) != 0 {
		t.Fatalf("unexpected major label/sort")
	}
	if GroupLabel(mMinor) != "Minor" || GroupSortKey(mMinor) != 1 {
		t.Fatalf("unexpected minor label/sort")
	}
	if GroupLabel(mPatch) != "Patch" || GroupSortKey(mPatch) != 2 {
		t.Fatalf("unexpected patch label/sort")
	}
	if GroupLabel(mV0Minor) != "Major (v0)" || GroupSortKey(mV0Minor) != 0 {
		t.Fatalf("unexpected v0 label/sort")
	}
}
