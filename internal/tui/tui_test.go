package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticivan/go-check-updates/internal/scanner"
)

func TestModelSelectionAndCursor(t *testing.T) {
	direct := []scanner.Module{{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}}}
	indirect := []scanner.Module{{Path: "b", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.0.1"}}}
	m := initialModel(direct, indirect, nil, Options{})

	// Move down
	modelAny, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := modelAny.(model)
	if m2.cursor != 1 {
		t.Fatalf("expected cursor=1, got %d", m2.cursor)
	}

	// Toggle selection with space
	modelAny, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m3 := modelAny.(model)
	if _, ok := m3.selected[1]; !ok {
		t.Fatalf("expected item selected")
	}

	// Toggle again
	modelAny, _ = m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m4 := modelAny.(model)
	if _, ok := m4.selected[1]; ok {
		t.Fatalf("expected item deselected")
	}
}

func TestInit_ReturnsNil(t *testing.T) {
	m := initialModel(nil, nil, nil, Options{})
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init cmd")
	}
}

func TestInitialModel_SortsWhenFormatGroup(t *testing.T) {
	// Use paths that should end up sorted by group and then path.
	direct := []scanner.Module{
		{Path: "b", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.0.1"}},
		{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v2.0.0"}},
	}
	m := initialModel(direct, nil, nil, Options{FormatGroup: true})
	if len(m.choices) != 2 {
		t.Fatalf("expected 2 choices")
	}
	// Major group should come first.
	if m.choices[0].Path != "a" {
		t.Fatalf("expected major update first, got %q", m.choices[0].Path)
	}
}

func TestViewContainsHeadings(t *testing.T) {
	direct := []scanner.Module{{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}}}
	m := initialModel(direct, nil, nil, Options{})
	view := m.View()
	if !strings.Contains(view, "Direct dependencies (go.mod)") {
		t.Fatalf("expected direct heading")
	}
	if !strings.Contains(view, "Which packages would you like to update?") {
		t.Fatalf("expected prompt")
	}
}

func TestStartInteractiveGroupedWithOptions_AppliesSelection(t *testing.T) {
	origRun := runProgram
	origUpdate := updatePackages
	defer func() {
		runProgram = origRun
		updatePackages = origUpdate
	}()

	direct := []scanner.Module{{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}}}
	base := initialModel(direct, nil, nil, Options{})
	base.selected[0] = struct{}{}

	called := false
	runProgram = func(tea.Model) (tea.Model, error) {
		return base, nil
	}
	updatePackages = func(mods []scanner.Module) error {
		called = true
		if len(mods) != 1 || mods[0].Path != "a" {
			t.Fatalf("unexpected modules: %#v", mods)
		}
		return nil
	}

	StartInteractiveGroupedWithOptions(direct, nil, nil, Options{})
	if !called {
		t.Fatalf("expected updatePackages to be called")
	}
}

func TestStartInteractiveGrouped_BackCompat(t *testing.T) {
	origRun := runProgram
	defer func() { runProgram = origRun }()

	runProgram = func(tea.Model) (tea.Model, error) {
		return initialModel(nil, nil, nil, Options{}), nil
	}
	StartInteractiveGrouped(nil, nil, nil)
}
