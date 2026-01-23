// Package npm provides npm package manager scanning functionality.
package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pragmaticivan/faro/internal/cooldown"
	"github.com/pragmaticivan/faro/internal/scanner"
)

// Scanner implements scanner.Scanner for npm.
type Scanner struct {
	workDir          string
	runNpmOutdated   func() ([]byte, error)
	fetchPackageTime func(name, version string) (string, error)
}

// packageJSON represents the structure of package.json.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// npmOutdated represents the structure of `npm outdated --json` output.
type npmOutdated map[string]npmPackageInfo

type npmPackageInfo struct {
	Current  string `json:"current"`
	Wanted   string `json:"wanted"`
	Latest   string `json:"latest"`
	Location string `json:"location"`
	Type     string `json:"type"` // "dependencies" or "devDependencies"
}

// NewScanner creates a new npm scanner.
func NewScanner(workDir string) *Scanner {
	s := &Scanner{
		workDir: workDir,
		runNpmOutdated: func() ([]byte, error) {
			cmd := exec.Command("npm", "outdated", "--json")
			cmd.Dir = workDir
			// npm outdated returns exit code 1 when there are outdated packages
			// So we ignore the error and just get the output
			out, _ := cmd.Output()
			return out, nil
		},
	}
	s.fetchPackageTime = func(name, version string) (string, error) {
		// npm view package time --json
		// Note: 'npm view' returns the full time map even if we ask for a specific version,
		// so we ask for the package time map and extract the specific version.
		cmd := exec.Command("npm", "view", name, "time", "--json")
		cmd.Dir = workDir
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}

		var timeMap map[string]string
		if err := json.Unmarshal(out, &timeMap); err != nil {
			return "", err
		}

		if t, ok := timeMap[version]; ok {
			return t, nil
		}
		return "", nil
	}
	return s
}

// GetUpdates returns all npm packages that have available updates.
func (s *Scanner) GetUpdates(opts scanner.Options) ([]scanner.Module, error) {
	// Read package.json to determine dependency types
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	// Get outdated packages from npm
	output, err := s.runNpmOutdated()
	if err != nil {
		return nil, fmt.Errorf("failed to run npm outdated: %w", err)
	}

	if len(output) == 0 {
		return []scanner.Module{}, nil
	}

	var outdated npmOutdated
	if err := json.Unmarshal(output, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse npm outdated output: %w", err)
	}

	type candidate struct {
		Name   string
		Info   npmPackageInfo
		Direct bool
		Type   string
	}
	var candidates []candidate

	for name, info := range outdated {
		// If current version matches latest, it's not an update we care about
		if info.Current == info.Latest {
			continue
		}

		// Determine if it's a direct dependency
		_, isDirect := pkgJSON.Dependencies[name]
		_, isDevDirect := pkgJSON.DevDependencies[name]

		depType := info.Type
		if depType == "" {
			if isDirect {
				depType = "dependencies"
			} else if isDevDirect {
				depType = "devDependencies"
			} else {
				depType = "transitive"
			}
		}

		// Filter devDependencies if not including all
		if !opts.IncludeAll && depType == "devDependencies" {
			continue
		}

		// Apply filter
		if opts.Filter != "" && !strings.Contains(name, opts.Filter) {
			continue
		}

		candidates = append(candidates, candidate{name, info, isDirect || isDevDirect, depType})
	}

	// Fetch update times concurrently
	modules := make([]scanner.Module, 0, len(candidates))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit concurrency

	for _, c := range candidates {
		wg.Add(1)
		go func(c candidate) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire token
			defer func() { <-sem }() // Release token

			var updateTime string
			// Only fetch time if we have a latest version
			if c.Info.Latest != "" {
				t, err := s.fetchPackageTime(c.Name, c.Info.Latest)
				if err == nil {
					updateTime = t
				}
			}

			// Apply cooldown if requested and we have a time
			if opts.CooldownDays > 0 && updateTime != "" {
				if !cooldown.Eligible(updateTime, opts.CooldownDays, time.Now()) {
					return
				}
			}

			module := scanner.Module{
				Name:           c.Name,
				Version:        c.Info.Current,
				Direct:         c.Direct,
				DependencyType: c.Type,
				Update: &scanner.UpdateInfo{
					Version: c.Info.Latest,
					Time:    updateTime,
				},
			}

			mu.Lock()
			modules = append(modules, module)
			mu.Unlock()
		}(c)
	}

	wg.Wait()
	return modules, nil
}

// GetDependencyIndex returns a map of npm package names to their dependency information.
func (s *Scanner) GetDependencyIndex() (scanner.DependencyIndex, error) {
	pkgJSON, err := s.readPackageJSON()
	if err != nil {
		return nil, err
	}

	idx := make(scanner.DependencyIndex)
	for name := range pkgJSON.Dependencies {
		idx[name] = scanner.DependencyInfo{
			Direct: true,
			Type:   "dependencies",
		}
	}
	for name := range pkgJSON.DevDependencies {
		idx[name] = scanner.DependencyInfo{
			Direct: true,
			Type:   "devDependencies",
		}
	}
	return idx, nil
}

// readPackageJSON reads and parses package.json.
func (s *Scanner) readPackageJSON() (*packageJSON, error) {
	path := filepath.Join(s.workDir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
