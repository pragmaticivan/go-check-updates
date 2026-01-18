package app

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pragmaticivan/go-check-updates/internal/scanner"
	"github.com/pragmaticivan/go-check-updates/internal/tui"
)

func TestRun_FormatLines_NoBanners(t *testing.T) {
	var out bytes.Buffer
	fixedNow := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)

	mods := []scanner.Module{
		{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}, FromGoMod: true},
		{Path: "b", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.0.1"}, FromGoMod: true, Indirect: true},
	}

	err := Run(RunOptions{FormatFlag: "lines"}, Deps{
		Out: &out,
		Now: func() time.Time { return fixedNow },
		GetUpdates: func(scanner.Options) ([]scanner.Module, error) {
			return mods, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "Checking for updates") {
		t.Fatalf("did not expect banners in lines format: %q", got)
	}
	if !strings.Contains(got, "a@v1.1.0") || !strings.Contains(got, "b@v1.0.1") {
		t.Fatalf("expected module lines, got: %q", got)
	}
}

func TestRun_Interactive_CallsHook(t *testing.T) {
	var out bytes.Buffer
	called := false
	mods := []scanner.Module{{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}, FromGoMod: true}}

	err := Run(RunOptions{Interactive: true}, Deps{
		Out:        &out,
		GetUpdates: func(scanner.Options) ([]scanner.Module, error) { return mods, nil },
		StartInteractive: func(d, i, tr []scanner.Module, _ tui.Options) {
			called = true
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatalf("expected interactive hook to be called")
	}
}

func TestRun_BadFormatFlag(t *testing.T) {
	var out bytes.Buffer
	err := Run(RunOptions{FormatFlag: "nope"}, Deps{Out: &out, GetUpdates: func(scanner.Options) ([]scanner.Module, error) {
		return nil, nil
	}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRun_NoUpdates_PrintsMessage(t *testing.T) {
	var out bytes.Buffer
	err := Run(RunOptions{}, Deps{
		Out: &out,
		GetUpdates: func(scanner.Options) ([]scanner.Module, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out.String(), "All dependencies match") {
		t.Fatalf("expected up-to-date message, got: %q", out.String())
	}
}

func TestRun_Upgrade_CallsUpdatePackages(t *testing.T) {
	var out bytes.Buffer
	called := false
	mods := []scanner.Module{{Path: "a", Version: "v1.0.0", Update: &scanner.Module{Version: "v1.1.0"}, FromGoMod: true}}
	err := Run(RunOptions{Upgrade: true}, Deps{
		Out:        &out,
		GetUpdates: func(scanner.Options) ([]scanner.Module, error) { return mods, nil },
		UpdatePackages: func(ms []scanner.Module) error {
			called = true
			if len(ms) != 1 || ms[0].Path != "a" {
				t.Fatalf("unexpected update list: %#v", ms)
			}
			return nil
		},
		StartInteractive: func(_, _, _ []scanner.Module, _ tui.Options) {},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatalf("expected UpdatePackages to be called")
	}
}

func TestRun_GroupedOutput_PrintsHeadings(t *testing.T) {
	var out bytes.Buffer
	fixedNow := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)
	mods := []scanner.Module{{
		Path:      "a",
		Version:   "v1.0.0",
		Update:    &scanner.Module{Version: "v1.0.1", Time: "2026-01-10T00:00:00Z"},
		FromGoMod: true,
	}}

	err := Run(RunOptions{FormatFlag: "group,time"}, Deps{
		Out:              &out,
		Now:              func() time.Time { return fixedNow },
		GetUpdates:       func(scanner.Options) ([]scanner.Module, error) { return mods, nil },
		StartInteractive: func(_, _, _ []scanner.Module, _ tui.Options) {},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "Available updates") || !strings.Contains(text, "Direct dependencies") {
		t.Fatalf("expected headings, got: %q", text)
	}
}
