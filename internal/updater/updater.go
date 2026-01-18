// Package updater handles the actual upgrading of Go modules using 'go get'.
package updater

import (
	"fmt"
	"os/exec"

	"github.com/pragmaticivan/go-check-updates/internal/scanner"
)

var runCombinedOutput = func(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func buildGoGetArgs(modules []scanner.Module) []string {
	args := []string{"get"}
	for _, m := range modules {
		if m.Update != nil && m.Update.Version != "" {
			args = append(args, fmt.Sprintf("%s@%s", m.Path, m.Update.Version))
			continue
		}
		args = append(args, m.Path)
	}
	return args
}

func UpdatePackages(modules []scanner.Module) error {
	if len(modules) == 0 {
		return nil
	}

	fmt.Printf("Upgrading %d packages...\n", len(modules))
	args := buildGoGetArgs(modules)
	if out, err := runCombinedOutput("go", args...); err != nil {
		return fmt.Errorf("go get failed: %s: %w", string(out), err)
	}

	// Tidy up
	if out, err := runCombinedOutput("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %s: %w", string(out), err)
	}

	return nil
}
