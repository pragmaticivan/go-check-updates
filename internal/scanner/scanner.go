// Package scanner handles the discovery of Go module updates by parsing 'go list' output.
package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pragmaticivan/go-check-updates/internal/cooldown"
	"github.com/pragmaticivan/go-check-updates/internal/gomod"
)

var goListAllModulesOutput = func() ([]byte, error) {
	cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
	return cmd.Output()
}

// Module represents a single Go module with version info
type Module struct {
	Path     string  `json:"Path"`
	Version  string  `json:"Version"`
	Time     string  `json:"Time"`
	Update   *Module `json:"Update"` // If there is an update, this struct is populated
	Indirect bool    `json:"Indirect"`

	// FromGoMod indicates this module is explicitly listed in go.mod.
	// It is populated by gcu (not by `go list`).
	FromGoMod bool `json:"-"`

	// VulnCurrent holds vulnerability counts for the current version
	VulnCurrent VulnInfo `json:"-"`
	// VulnUpdate holds vulnerability counts for the update version
	VulnUpdate VulnInfo `json:"-"`
}

// VulnInfo contains vulnerability information for a module version
type VulnInfo struct {
	Low      int
	Medium   int
	High     int
	Critical int
	Total    int
}

// Options configures dependency discovery.
type Options struct {
	Filter       string
	FilterRegex  *regexp.Regexp
	IncludeAll   bool
	CooldownDays int
}

// DecodeGoListModules decodes the JSON stream output from:
//
//	go list -m -u -json all
func DecodeGoListModules(data []byte) ([]Module, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var modules []Module
	for decoder.More() {
		var m Module
		if err := decoder.Decode(&m); err != nil {
			return nil, fmt.Errorf("failed to decode json: %w", err)
		}
		modules = append(modules, m)
	}
	return modules, nil
}

// AnnotateAndFilter applies go.mod classification and filters modules based on opts.
func AnnotateAndFilter(modules []Module, idx gomod.RequireIndex, opts Options, now time.Time) []Module {
	out := make([]Module, 0, len(modules))
	for _, m := range modules {
		if m.Update == nil {
			continue
		}

		// Override classification based on go.mod
		if indirect, ok := idx[m.Path]; ok {
			m.FromGoMod = true
			m.Indirect = indirect
		}

		if !opts.IncludeAll && !m.FromGoMod {
			continue
		}
		if opts.Filter != "" {
			match := strings.Contains(m.Path, opts.Filter)
			if !match && opts.FilterRegex != nil {
				match = opts.FilterRegex.MatchString(m.Path)
			}
			if !match {
				continue
			}
		}
		if opts.CooldownDays > 0 {
			if !cooldown.Eligible(m.Update.Time, opts.CooldownDays, now) {
				continue
			}
		}

		out = append(out, m)
	}
	return out
}

// GetUpdates runs 'go list' to find available updates.
//
// By default (includeAll=false), it only returns updates for modules explicitly listed in go.mod,
// splitable into direct vs indirect as described in go.mod.
//
// When includeAll=true, it returns updates for all modules (including transitive), and still
// annotates any that are explicitly listed in go.mod.
func GetUpdates(opts Options) ([]Module, error) {
	return GetUpdatesFrom(filepath.Join(".", "go.mod"), opts)
}

// GetUpdatesFrom finds updates using the go.mod at goModPath.
// This is primarily useful for testing and advanced callers.
func GetUpdatesFrom(goModPath string, opts Options) ([]Module, error) {
	idx, err := gomod.ReadRequireIndex(goModPath)
	if err != nil {
		return nil, err
	}

	if opts.Filter != "" && opts.FilterRegex == nil {
		compiled, err := regexp.Compile(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter pattern: %w", err)
		}
		opts.FilterRegex = compiled
	}

	output, err := goListAllModulesOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run go list: %w", err)
	}

	modules, err := DecodeGoListModules(output)
	if err != nil {
		return nil, err
	}
	return AnnotateAndFilter(modules, idx, opts, time.Now()), nil
}
