package updater

import (
	"reflect"
	"testing"

	"github.com/pragmaticivan/go-check-updates/internal/scanner"
)

func TestBuildGoGetArgs(t *testing.T) {
	mods := []scanner.Module{
		{Path: "example.com/a", Update: &scanner.Module{Version: "v1.2.3"}},
		{Path: "example.com/b", Update: &scanner.Module{Version: ""}},
		{Path: "example.com/c"},
	}

	got := buildGoGetArgs(mods)
	want := []string{"get", "example.com/a@v1.2.3", "example.com/b", "example.com/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args\n got: %#v\nwant: %#v", got, want)
	}
}

func TestUpdatePackages_RunsGoGetThenTidy(t *testing.T) {
	orig := runCombinedOutput
	defer func() { runCombinedOutput = orig }()

	var calls [][]string
	runCombinedOutput = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte("ok"), nil
	}

	mods := []scanner.Module{{Path: "example.com/a", Update: &scanner.Module{Version: "v1.2.3"}}}
	if err := UpdatePackages(mods); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0][0] != "go" || calls[0][1] != "get" {
		t.Fatalf("expected go get first, got %#v", calls[0])
	}
	if calls[1][0] != "go" || calls[1][1] != "mod" || calls[1][2] != "tidy" {
		t.Fatalf("expected go mod tidy second, got %#v", calls[1])
	}
}
