package style

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/pragmaticivan/go-check-updates/internal/scanner"
)

func init() {
	// Force color profile to ANSI 256 or TrueColor
	lipgloss.SetColorProfile(termenv.ANSI256)

	ColorMajor = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))          // Red
	ColorMinor = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))           // Cyan
	ColorPatch = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))           // Green
	ColorReset = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))          // Light Gray/White
	ColorPath = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true) // Cyan Bold (nc style)
	ColorArrow = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))          // Grey
	ColorUnknown = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))         // Magenta
}

type DiffType int

const (
	DiffMajor DiffType = iota
	DiffMinor
	DiffPatch
	DiffSame
	DiffUnknown
)

// Define colors
var (
	ColorMajor   lipgloss.Style
	ColorMinor   lipgloss.Style
	ColorPatch   lipgloss.Style
	ColorReset   lipgloss.Style
	ColorPath    lipgloss.Style
	ColorArrow   lipgloss.Style
	ColorUnknown lipgloss.Style
)

func GetDiffType(v1, v2 string) DiffType {
	if isPseudoVersion(v1) || isPseudoVersion(v2) {
		return DiffUnknown
	}
	if v1 == v2 {
		return DiffSame
	}

	// Try to compare semver-like module versions (vMAJOR.MINOR.PATCH with optional -prerelease/+meta).
	// Pseudo-versions and other non-standard forms fall back to unknown.
	ma1, mi1, pa1, ok1 := parseSemverCore(v1)
	ma2, mi2, pa2, ok2 := parseSemverCore(v2)
	if !ok1 || !ok2 {
		return DiffUnknown
	}

	if ma1 != ma2 {
		return DiffMajor
	}
	if mi1 != mi2 {
		return DiffMinor
	}
	if pa1 != pa2 {
		return DiffPatch
	}
	return DiffSame
}

func parseSemverCore(v string) (major, minor, patch int, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, 0, 0, false
	}
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}

	ma, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	mi, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	pa, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}
	if ma < 0 || mi < 0 || pa < 0 {
		return 0, 0, 0, false
	}
	return ma, mi, pa, true
}

func isPseudoVersion(v string) bool {
	// Go pseudo versions always contain two hyphen-separated suffix segments,
	// e.g. v1.2.3-20240101000000-abcdef123456.
	return strings.Count(v, "-") >= 2
}

func GetVersionStyle(diff DiffType) lipgloss.Style {
	switch diff {
	case DiffMajor:
		return ColorMajor
	case DiffMinor:
		return ColorMinor
	case DiffPatch:
		return ColorPatch
	case DiffUnknown:
		return ColorUnknown
	default:
		return ColorReset
	}
}

// FormatUpdate returns a colored string: "package  v1.0.0 -> v2.0.0" (with colors)
// paddedPath needs to be calculated by the caller for alignment
func FormatUpdate(path, vOld, vNew string, padPath int) string {
	diff := GetDiffType(vOld, vNew)
	targetStyle := GetVersionStyle(diff)

	// Format: PATH (cyan)  vOld (white)  -> (grey)  vNew (colored)

	// Ensure padding
	pPath := fmt.Sprintf("%-*s", padPath, path)

	return fmt.Sprintf("%s  %s  %s  %s",
		ColorPath.Render(pPath),
		vOld,
		ColorArrow.Render("→"),
		targetStyle.Render(vNew),
	)
}

// FormatUpdateWithVulns formats a module update line with vulnerability information
func FormatUpdateWithVulns(path, vOld, vNew string, padPath int, vulnCurrent, vulnUpdate scanner.VulnInfo, showVulns bool) string {
	diff := GetDiffType(vOld, vNew)
	targetStyle := GetVersionStyle(diff)

	// Ensure padding
	pPath := fmt.Sprintf("%-*s", padPath, path)

	// Format vulnerability counts
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	orange := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	formatVulnInfo := func(info scanner.VulnInfo) string {
		if info.Total == 0 {
			return ""
		}

		parts := []string{}
		if info.Low > 0 {
			parts = append(parts, fmt.Sprintf("L (%d)", info.Low))
		}
		if info.Medium > 0 {
			parts = append(parts, yellow.Render(fmt.Sprintf("M (%d)", info.Medium)))
		}
		if info.High > 0 {
			parts = append(parts, orange.Render(fmt.Sprintf("H (%d)", info.High)))
		}
		if info.Critical > 0 {
			parts = append(parts, red.Render(fmt.Sprintf("C (%d)", info.Critical)))
		}

		if len(parts) == 0 {
			return ""
		}

		result := " ["
		for i, part := range parts {
			if i > 0 {
				result += ", "
			}
			result += part
		}
		result += "]"
		return result
	}

	// Build the line
	line := fmt.Sprintf("%s  %s", ColorPath.Render(pPath), vOld)

	// Add current version vulnerabilities
	if showVulns && vulnCurrent.Total > 0 {
		line += formatVulnInfo(vulnCurrent)
	}

	line += "  " + ColorArrow.Render("→") + "  " + targetStyle.Render(vNew)

	// Add update version vulnerabilities or fixed indicator
	if showVulns && vulnCurrent.Total > 0 {
		fixed := vulnCurrent.Total - vulnUpdate.Total

		if fixed > 0 {
			// Vulnerabilities were fixed
			if vulnUpdate.Total == 0 {
				line += " " + green.Render(fmt.Sprintf("✓ (fixes %d)", fixed))
			} else {
				line += formatVulnInfo(vulnUpdate) + " " + green.Render(fmt.Sprintf("(fixes %d)", fixed))
			}
		} else if fixed < 0 {
			// More vulnerabilities in update
			line += formatVulnInfo(vulnUpdate) + " " + red.Render(fmt.Sprintf("(+%d)", -fixed))
		} else if vulnUpdate.Total > 0 {
			// Same count but might be different types
			line += formatVulnInfo(vulnUpdate)
		}
	}

	return line
}
