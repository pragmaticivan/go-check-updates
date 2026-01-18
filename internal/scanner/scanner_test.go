package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pragmaticivan/go-check-updates/internal/gomod"
)

func TestDecodeGoListModules(t *testing.T) {
	data := []byte(`{"Path":"a","Version":"v1.0.0","Update":{"Version":"v1.1.0","Time":"2020-01-01T00:00:00Z"}}{"Path":"b","Version":"v1.0.0"}`)
	mods, err := DecodeGoListModules(data)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(mods) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(mods))
	}
	if mods[0].Path != "a" || mods[1].Path != "b" {
		t.Fatalf("unexpected paths: %#v", mods)
	}
}

func TestAnnotateAndFilter(t *testing.T) {
	now := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	mods := []Module{
		{Path: "direct", Version: "v1.0.0", Update: &Module{Version: "v1.0.1", Time: old}},
		{Path: "trans", Version: "v1.0.0", Update: &Module{Version: "v1.0.1", Time: old}},
		{Path: "noupdate", Version: "v1.0.0"},
	}
	idx := gomod.RequireIndex{"direct": false}

	out := AnnotateAndFilter(mods, idx, Options{IncludeAll: false, CooldownDays: 30}, now)
	if len(out) != 1 {
		t.Fatalf("expected 1 module, got %d", len(out))
	}
	if !out[0].FromGoMod || out[0].Indirect {
		t.Fatalf("expected direct module to be FromGoMod and direct")
	}

	outAll := AnnotateAndFilter(mods, idx, Options{IncludeAll: true, Filter: "tr"}, now)
	if len(outAll) != 1 || outAll[0].Path != "trans" {
		t.Fatalf("expected filtered transitive module, got %#v", outAll)
	}
}

func TestGetUpdatesFrom_UsesGoModAndInjectedGoList(t *testing.T) {
	orig := goListAllModulesOutput
	defer func() { goListAllModulesOutput = orig }()

	now := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	old := now.Add(-60 * 24 * time.Hour).Format(time.RFC3339)

	// Create a temp go.mod
	dir := t.TempDir()
	goModPath := filepath.Join(dir, "go.mod")
	goMod := `module example.com/foo

go 1.25

require (
	go.mod/direct v1.0.0
	go.mod/indirect v1.0.0 // indirect
)
`
	if err := os.WriteFile(goModPath, []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Inject fake `go list` JSON stream.
	goListAllModulesOutput = func() ([]byte, error) {
		return []byte(
			`{"Path":"go.mod/direct","Version":"v1.0.0","Update":{"Version":"v1.0.1","Time":"` + old + `"}}` +
				`{"Path":"go.mod/indirect","Version":"v1.0.0","Update":{"Version":"v1.1.0","Time":"` + old + `"}}` +
				`{"Path":"transitive","Version":"v1.0.0","Update":{"Version":"v1.0.1","Time":"` + old + `"}}`,
		), nil
	}

	// Default: includeAll=false should only return go.mod modules.
	mods, err := GetUpdatesFrom(goModPath, Options{IncludeAll: false, CooldownDays: 30})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(mods) != 2 {
		t.Fatalf("expected 2 go.mod modules, got %d", len(mods))
	}
	if !mods[0].FromGoMod || !mods[1].FromGoMod {
		t.Fatalf("expected FromGoMod=true")
	}

	modsAll, err := GetUpdatesFrom(goModPath, Options{IncludeAll: true, CooldownDays: 30})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(modsAll) != 3 {
		t.Fatalf("expected 3 modules with includeAll=true, got %d", len(modsAll))
	}
}

func TestGetUpdates_DefaultPath(t *testing.T) {
	origOut := goListAllModulesOutput
	defer func() { goListAllModulesOutput = origOut }()

	dir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	goMod := `module example.com/foo

go 1.25

require example.com/a v1.0.0
`
	if err := os.WriteFile("go.mod", []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goListAllModulesOutput = func() ([]byte, error) {
		return []byte(`{"Path":"example.com/a","Version":"v1.0.0","Update":{"Version":"v1.0.1","Time":"2020-01-01T00:00:00Z"}}`), nil
	}

	mods, err := GetUpdates(Options{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 module, got %d", len(mods))
	}
}
