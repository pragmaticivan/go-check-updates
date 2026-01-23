package gomod

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pragmaticivan/faro/internal/scanner"
)

func TestGetUpdates(t *testing.T) {
	// 1. Setup go.mod
	tmpDir := t.TempDir()
	goModContent := `
module example.com/foo

go 1.21

require (
	example.com/direct v1.0.0
	example.com/indirect v1.0.0 // indirect
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// 2. Setup mock go list output
	mockOutput := []goModule{
		{
			Path:     "example.com/direct",
			Version:  "v1.0.0",
			Indirect: false,
			Update: &goModule{
				Path:    "example.com/direct",
				Version: "v1.2.0",
				Time:    "2023-01-01T00:00:00Z",
			},
		},
		{
			Path:     "example.com/indirect",
			Version:  "v1.0.0",
			Indirect: true,
			Update: &goModule{
				Path:    "example.com/indirect",
				Version: "v1.1.0",
				Time:    "2023-01-01T00:00:00Z",
			},
		},
		{
			Path:     "example.com/transitive",
			Version:  "v0.5.0",
			Indirect: true,
			Update: &goModule{
				Path:    "example.com/transitive",
				Version: "v0.6.0",
				Time:    "2023-01-01T00:00:00Z",
			},
		},
	}

	listOutput := []byte{}
	for _, m := range mockOutput {
		b, _ := json.Marshal(m)
		listOutput = append(listOutput, b...)
	}

	// 3. Initialize Scanner
	s := NewScanner(tmpDir)
	s.listAllModules = func() ([]byte, error) {
		// go list -json output is a stream of JSON objects, not an array
		var buf []byte
		for _, m := range mockOutput {
			b, _ := json.Marshal(m)
			buf = append(buf, b...)
		}
		return buf, nil
	}

	// 4. Test Case: Default options (Direct + Indirect in go.mod, no transitive that aren't in go.mod)
	// Wait, the logic is:
	// if includeAll is false, we skip if !fromGoMod.
	// direct is in go.mod (direct=true)
	// indirect is in go.mod (direct=false)
	// transitive is NOT in go.mod

	opts := scanner.Options{
		IncludeAll: false,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	// Should have direct and indirect, but NOT transitive
	// Check count
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}

	// Verify "example.com/direct" is Direct=true
	foundDirect := false
	for _, m := range modules {
		if m.Name == "example.com/direct" {
			foundDirect = true
			if !m.Direct {
				t.Error("expected example.com/direct to be Direct=true")
			}
			if m.DependencyType != "direct" {
				t.Errorf("expected dependency type 'direct', got %s", m.DependencyType)
			}
		}
		if m.Name == "example.com/indirect" {
			if m.Direct {
				t.Error("expected example.com/indirect to be Direct=false")
			}
			if m.DependencyType != "indirect" {
				t.Errorf("expected dependency type 'indirect', got %s", m.DependencyType)
			}
		}
	}
	if !foundDirect {
		t.Error("example.com/direct not found")
	}

	// 5. Test Case: IncludeAll = true
	opts.IncludeAll = true
	modules, err = s.GetUpdates(opts)
	if err != nil {
		t.Fatalf("GetUpdates(IncludeAll) failed: %v", err)
	}
	if len(modules) != 3 {
		t.Errorf("expected 3 modules with IncludeAll, got %d", len(modules))
	}
}

func TestGetUpdates_Cooldown(t *testing.T) {
	tmpDir := t.TempDir()
	goModContent := `module test
go 1.21
require example.com/pkg v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	now := time.Now()
	freshTime := now.Add(-1 * time.Hour).Format(time.RFC3339) // 1 hour ago
	oldTime := now.Add(-48 * time.Hour).Format(time.RFC3339)  // 2 days ago

	mockOutput := []goModule{
		{
			Path: "example.com/pkg", Version: "v1.0.0",
			Update: &goModule{Path: "example.com/pkg", Version: "v1.1.0", Time: freshTime},
		},
		{
			Path: "example.com/old", Version: "v1.0.0", Indirect: true, // Treated as direct because go.mod? No, go.mod only has pkg.
			// Wait, let's add checking logic.
			// If not in go.mod, it's skipped unless IncludeAll is true.
			// Let's add "example.com/old" to go.mod to make it simpler, or use IncludeAll.
			Update: &goModule{Path: "example.com/old", Version: "v1.1.0", Time: oldTime},
		},
	}
	// Add pkg to go.mod above.

	// Create scanner
	s := NewScanner(tmpDir)
	s.listAllModules = func() ([]byte, error) {
		var buf []byte
		for _, m := range mockOutput {
			b, _ := json.Marshal(m)
			buf = append(buf, b...)
		}
		return buf, nil
	}

	// Case 1: Cooldown 1 day. Fresh should be skipped. Old (48h) should pass.
	// But "example.com/old" is not in go.mod, so it's skipped by default.
	// Let's rely on IncludeAll for the second package or assume it's in go.mod?
	// The test setup only put pkg in go.mod. So 'old' is not in go.mod.
	// Let's use IncludeAll = true to test cooldown on both.

	opts := scanner.Options{
		CooldownDays: 1,
		IncludeAll:   true,
	}

	modules, err := s.GetUpdates(opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(modules) != 1 {
		t.Errorf("expected 1 module (old), got %d", len(modules))
	} else {
		if modules[0].Name != "example.com/old" {
			t.Errorf("expected example.com/old, got %s", modules[0].Name)
		}
	}
}

func TestDecodeGoListModules(t *testing.T) {
	input := `
{
	"Path": "example.com/a",
	"Version": "v1.0.0"
}
{
	"Path": "example.com/b",
	"Version": "v2.0.0"
}
`
	modules, err := decodeGoListModules([]byte(input))
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}
	if modules[0].Path != "example.com/a" {
		t.Errorf("first module is %s", modules[0].Path)
	}
	if modules[1].Path != "example.com/b" {
		t.Errorf("second module is %s", modules[1].Path)
	}
}

// Helper struct field need 'Refresh' was a typo in my mind?
// No, goModule struct in scanner.go doesn't have Refresh. I added it in the test mock struct init but it's not in the type definition in scanner.go.
// I need to be careful. The mock is creating goModule structs.
