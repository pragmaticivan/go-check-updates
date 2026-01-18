package style

import (
	"strings"
	"testing"
)

func TestGetDiffType_Semver(t *testing.T) {
	if GetDiffType("v1.2.3", "v2.0.0") != DiffMajor {
		t.Fatalf("expected major")
	}
	if GetDiffType("v1.2.3", "v1.3.0") != DiffMinor {
		t.Fatalf("expected minor")
	}
	if GetDiffType("v1.2.3", "v1.2.4") != DiffPatch {
		t.Fatalf("expected patch")
	}
	if GetDiffType("v1.2.3", "v1.2.3") != DiffSame {
		t.Fatalf("expected same")
	}
}

func TestGetDiffType_NonStandard(t *testing.T) {
	// pseudo-versions should fall back to unknown
	if GetDiffType("v0.0.0-20240101000000-abcdef", "v0.0.0-20240102000000-bcdef0") != DiffUnknown {
		t.Fatalf("expected unknown for pseudo versions")
	}
	// pre-release should be handled by stripping suffix
	if GetDiffType("v1.2.3-beta.1", "v1.2.4") != DiffPatch {
		t.Fatalf("expected patch when core semver changes")
	}
}

func TestFormatUpdate_IncludesPathAndVersions(t *testing.T) {
	got := FormatUpdate("example.com/mod", "v1.0.0", "v1.0.1", 20)
	if got == "" {
		t.Fatalf("expected non-empty output")
	}
	if !strings.Contains(got, "example.com/mod") || !strings.Contains(got, "v1.0.0") || !strings.Contains(got, "v1.0.1") {
		t.Fatalf("unexpected formatted output: %q", got)
	}
	if !strings.Contains(got, "→") {
		t.Fatalf("expected arrow in output: %q", got)
	}
}

func TestGetVersionStyle_DoesNotPanic(t *testing.T) {
	_ = GetVersionStyle(DiffMajor)
	_ = GetVersionStyle(DiffMinor)
	_ = GetVersionStyle(DiffPatch)
	_ = GetVersionStyle(DiffUnknown)
	_ = GetVersionStyle(DiffSame)
}
