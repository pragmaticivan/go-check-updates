package npm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestGetUpdates_WithTime(t *testing.T) {
	// Mock package.json data
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"react": "^18.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)

	// Mock npm outdated output
	mockOutdated := npmOutdated{
		"react": npmPackageInfo{
			Current: "18.0.0",
			Latest:  "18.2.0",
			Type:    "dependencies",
		},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		workDir: ".",
		// Overriding readPackageJSON needs logic, but we can't easily override a private method
		// that reads from disk in this design without more refactoring.
		// However, we can mock runNpmOutdated.
		// For readPackageJSON, we might need to rely on a file or refactor separation.
		// Wait, NewScanner takes workDir. We can create a temp dir and write package.json there.
		runNpmOutdated: func() ([]byte, error) {
			return outdatedBytes, nil
		},
		fetchPackageTime: func(name, version string) (string, error) {
			if name == "react" && version == "18.2.0" {
				return "2023-05-01T12:00:00.000Z", nil
			}
			return "", nil
		},
	}

	// Create temp directory for package.json
	tmpDir := t.TempDir()
	s.workDir = tmpDir

	// Write package.json
	if err := writePackageJSON(tmpDir, pkgJSONBytes); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	opts := scanner.Options{
		CooldownDays: 0,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}

	m := modules[0]
	if m.Name != "react" {
		t.Errorf("expected name react, got %s", m.Name)
	}
	if m.Update.Time != "2023-05-01T12:00:00.000Z" {
		t.Errorf("expected time 2023-05-01T12:00:00.000Z, got %s", m.Update.Time)
	}
}

func TestGetUpdates_Cooldown(t *testing.T) {
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"fresh-pkg": "^1.0.0",
			"old-pkg":   "^1.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)

	mockOutdated := npmOutdated{
		"fresh-pkg": {Current: "1.0.0", Latest: "2.0.0", Type: "dependencies"},
		"old-pkg":   {Current: "1.0.0", Latest: "2.0.0", Type: "dependencies"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		runNpmOutdated: func() ([]byte, error) {
			return outdatedBytes, nil
		},
		fetchPackageTime: func(name, version string) (string, error) {
			now := time.Now()
			if name == "fresh-pkg" {
				return now.Add(-24 * time.Hour).Format(time.RFC3339), nil // 1 day old
			}
			if name == "old-pkg" {
				return now.Add(-10 * 24 * time.Hour).Format(time.RFC3339), nil // 10 days old
			}
			return "", nil
		},
	}

	tmpDir := t.TempDir()
	s.workDir = tmpDir
	if err := writePackageJSON(tmpDir, pkgJSONBytes); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// 7 days cooldown
	opts := scanner.Options{CooldownDays: 7}
	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 1 {
		t.Fatalf("expected 1 module (old-pkg), got %d: %v", len(modules), modules)
	}
	if modules[0].Name != "old-pkg" {
		t.Errorf("expected old-pkg, got %s", modules[0].Name)
	}
}

func writePackageJSON(dir string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)
}

func TestGetUpdates_SkipSameVersion(t *testing.T) {
	mockPkgJSON := packageJSON{
		Dependencies: map[string]string{
			"up-to-date-pkg": "^1.0.0",
			"outdated-pkg":   "^1.0.0",
		},
	}
	pkgJSONBytes, _ := json.Marshal(mockPkgJSON)

	mockOutdated := npmOutdated{
		"up-to-date-pkg": {Current: "1.0.0", Latest: "1.0.0", Type: "dependencies"},
		"outdated-pkg":   {Current: "1.0.0", Latest: "2.0.0", Type: "dependencies"},
	}
	outdatedBytes, _ := json.Marshal(mockOutdated)

	s := &Scanner{
		runNpmOutdated: func() ([]byte, error) {
			return outdatedBytes, nil
		},
		fetchPackageTime: func(name, version string) (string, error) {
			return "", nil
		},
	}

	tmpDir := t.TempDir()
	s.workDir = tmpDir
	if err := writePackageJSON(tmpDir, pkgJSONBytes); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	modules, err := s.GetUpdates(scanner.Options{})
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}
	if modules[0].Name != "outdated-pkg" {
		t.Errorf("expected outdated-pkg, got %s", modules[0].Name)
	}
}

func TestParseNpmViewTime(t *testing.T) {
	// Simulate the output from npm view package time --json
	jsonOutput := `{
		"created": "2020-10-31T16:17:57.473Z",
		"modified": "2026-01-22T10:01:38.674Z",
		"0.11.0": "2020-10-31T16:17:58.512Z",
		"0.38.2": "2026-01-22T10:01:38.294Z"
	}`

	var timeMap map[string]string
	if err := json.Unmarshal([]byte(jsonOutput), &timeMap); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if val, ok := timeMap["0.38.2"]; !ok || val != "2026-01-22T10:01:38.294Z" {
		t.Errorf("expected 2026-01-22T10:01:38.294Z, got %s", val)
	}
}
