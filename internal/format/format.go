package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/pragmaticivan/go-check-updates/internal/scanner"
	"github.com/pragmaticivan/go-check-updates/internal/style"
)

type Options struct {
	Group bool
	Lines bool
	Time  bool
}

func ParseFlag(s string) (Options, error) {
	var out Options
	if strings.TrimSpace(s) == "" {
		return out, nil
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" {
			continue
		}
		switch v {
		case "group":
			out.Group = true
		case "lines":
			out.Lines = true
		case "time":
			out.Time = true
		default:
			return out, fmt.Errorf("unsupported --format value: %q (supported: group, lines, time)", v)
		}
	}
	return out, nil
}

func ParseRFC3339ish(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func PublishTime(updateTime string, now time.Time) string {
	t, ok := ParseRFC3339ish(updateTime)
	if !ok {
		return ""
	}
	days := int(now.Sub(t).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("%s (%dd ago)", t.Format("2006-01-02"), days)
}

type DiffGroup int

const (
	GroupMajor DiffGroup = iota
	GroupMinor
	GroupPatch
	GroupUnknown
)

func GroupForModule(m scanner.Module) DiffGroup {
	if m.Update == nil {
		return GroupUnknown
	}
	diff := style.GetDiffType(m.Version, m.Update.Version)
	switch diff {
	case style.DiffMajor:
		return GroupMajor
	case style.DiffMinor:
		// ncu-style behavior: v0 minor bumps are treated as major-ish risk
		if strings.HasPrefix(m.Version, "v0.") && strings.HasPrefix(m.Update.Version, "v0.") {
			return GroupMajor
		}
		return GroupMinor
	case style.DiffPatch:
		return GroupPatch
	default:
		return GroupUnknown
	}
}

func GroupLabel(m scanner.Module) string {
	if m.Update == nil {
		return "Unknown"
	}
	diff := style.GetDiffType(m.Version, m.Update.Version)
	if diff == style.DiffMajor {
		return "Major"
	}
	if diff == style.DiffMinor {
		if strings.HasPrefix(m.Version, "v0.") && strings.HasPrefix(m.Update.Version, "v0.") {
			return "Major (v0)"
		}
		return "Minor"
	}
	if diff == style.DiffPatch {
		return "Patch"
	}
	return "Unknown"
}

func GroupSortKey(m scanner.Module) int {
	switch GroupForModule(m) {
	case GroupMajor:
		return 0
	case GroupMinor:
		return 1
	case GroupPatch:
		return 2
	default:
		return 3
	}
}
