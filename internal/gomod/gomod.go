package gomod

import (
	"fmt"
	"os"
	"strings"
)

// RequireIndex maps module path -> indirect?
// A value of false means direct; true means listed with `// indirect`.
//
// If a module appears as both indirect and direct, direct wins.
type RequireIndex map[string]bool

func ReadRequireIndex(goModPath string) (RequireIndex, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", goModPath, err)
	}
	return ParseRequireIndex(string(data)), nil
}

func ParseRequireIndex(goModContents string) RequireIndex {
	idx := make(RequireIndex)

	lines := strings.Split(goModContents, "\n")
	inRequireBlock := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
			parseRequireLine(idx, line)
			continue
		}

		if inRequireBlock {
			parseRequireLine(idx, line)
		}
	}

	return idx
}

func parseRequireLine(dst RequireIndex, line string) {
	comment := ""
	if i := strings.Index(line, "//"); i >= 0 {
		comment = line[i+2:]
		line = strings.TrimSpace(line[:i])
	}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}

	path := fields[0]
	indirect := strings.Contains(comment, "indirect")

	if existingIndirect, ok := dst[path]; ok {
		if !existingIndirect {
			// already direct; keep direct
			dst[path] = false
			return
		}
		// previously indirect; upgrade to direct if we see a direct require
		dst[path] = indirect
		return
	}

	dst[path] = indirect
}
