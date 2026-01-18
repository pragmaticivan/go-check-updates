package gomod

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRequireIndex_BlockAndInline(t *testing.T) {
	contents := `module example.com/foo

go 1.25

require (
	github.com/a/b v1.2.3
	github.com/c/d v0.1.0 // indirect
)

require github.com/e/f v1.0.0 // indirect
`

	idx := ParseRequireIndex(contents)
	if idx["github.com/a/b"] != false {
		t.Fatalf("expected github.com/a/b to be direct")
	}
	if idx["github.com/c/d"] != true {
		t.Fatalf("expected github.com/c/d to be indirect")
	}
	if idx["github.com/e/f"] != true {
		t.Fatalf("expected github.com/e/f to be indirect")
	}
}

func TestParseRequireIndex_DirectWins(t *testing.T) {
	contents := `module example.com/foo

go 1.25

require (
	github.com/x/y v1.0.0 // indirect
	github.com/x/y v1.0.0
)
`

	idx := ParseRequireIndex(contents)
	if idx["github.com/x/y"] != false {
		t.Fatalf("expected github.com/x/y to be direct when both appear")
	}
}

func TestReadRequireIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	contents := "module example.com/foo\n\nrequire github.com/a/b v1.2.3\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	idx, err := ReadRequireIndex(path)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if idx["github.com/a/b"] != false {
		t.Fatalf("expected direct require")
	}
}
