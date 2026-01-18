package app

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/pragmaticivan/go-check-updates/internal/format"
	"github.com/pragmaticivan/go-check-updates/internal/scanner"
	"github.com/pragmaticivan/go-check-updates/internal/style"
	"github.com/pragmaticivan/go-check-updates/internal/tui"
)

type RunOptions struct {
	Upgrade     bool
	Interactive bool
	Filter      string
	All         bool
	Cooldown    int
	FormatFlag  string
}

type Deps struct {
	Out              io.Writer
	Now              func() time.Time
	GetUpdates       func(scanner.Options) ([]scanner.Module, error)
	UpdatePackages   func([]scanner.Module) error
	StartInteractive func(direct, indirect, transitive []scanner.Module, opts tui.Options)
}

func Run(opts RunOptions, deps Deps) error {
	if deps.Out == nil {
		return fmt.Errorf("missing deps.Out")
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.GetUpdates == nil {
		return fmt.Errorf("missing deps.GetUpdates")
	}

	formats, err := format.ParseFlag(opts.FormatFlag)
	if err != nil {
		return err
	}

	if !formats.Lines {
		fmt.Fprintln(deps.Out, "Checking for updates...")
	}

	modules, err := deps.GetUpdates(scanner.Options{
		Filter:       opts.Filter,
		IncludeAll:   opts.All,
		CooldownDays: opts.Cooldown,
	})
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		if !formats.Lines {
			fmt.Fprintln(deps.Out, "All dependencies match the latest package versions :)")
		}
		return nil
	}

	var direct, indirect, transitive []scanner.Module
	for _, m := range modules {
		if m.FromGoMod {
			if m.Indirect {
				indirect = append(indirect, m)
			} else {
				direct = append(direct, m)
			}
		} else {
			transitive = append(transitive, m)
		}
	}

	if opts.Interactive {
		if deps.StartInteractive == nil {
			return fmt.Errorf("missing deps.StartInteractive")
		}
		deps.StartInteractive(direct, indirect, transitive, tui.Options{
			FormatGroup: formats.Group,
			FormatTime:  formats.Time,
		})
		return nil
	}

	if formats.Lines {
		all := make([]scanner.Module, 0, len(direct)+len(indirect)+len(transitive))
		all = append(all, direct...)
		all = append(all, indirect...)
		if opts.All {
			all = append(all, transitive...)
		}
		for _, m := range all {
			if m.Update == nil {
				continue
			}
			fmt.Fprintf(deps.Out, "%s@%s\n", m.Path, m.Update.Version)
		}
		return nil
	}

	fmt.Fprintln(deps.Out, "\nAvailable updates:")

	maxPathLen := 0
	for _, group := range [][]scanner.Module{direct, indirect, transitive} {
		for _, m := range group {
			if len(m.Path) > maxPathLen {
				maxPathLen = len(m.Path)
			}
		}
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	printGroup := func(title string, group []scanner.Module) {
		if len(group) == 0 {
			return
		}
		fmt.Fprintf(deps.Out, "\n%s\n", title)

		if formats.Group {
			byLabel := make(map[string][]scanner.Module)
			order := make(map[string]int)
			for _, m := range group {
				label := format.GroupLabel(m)
				byLabel[label] = append(byLabel[label], m)
				if _, ok := order[label]; !ok {
					order[label] = format.GroupSortKey(m)
				}
			}
			labels := make([]string, 0, len(byLabel))
			for k := range byLabel {
				labels = append(labels, k)
			}
			sort.Slice(labels, func(i, j int) bool {
				if order[labels[i]] != order[labels[j]] {
					return order[labels[i]] < order[labels[j]]
				}
				return labels[i] < labels[j]
			})

			for _, label := range labels {
				fmt.Fprintf(deps.Out, "\n%s\n", dim.Render(label))
				for _, m := range byLabel[label] {
					line := " " + style.FormatUpdate(m.Path, m.Version, m.Update.Version, maxPathLen)
					if formats.Time {
						pt := format.PublishTime(m.Update.Time, deps.Now())
						if pt != "" {
							line += "  " + dim.Render(pt)
						}
					}
					fmt.Fprintln(deps.Out, line)
				}
			}
			return
		}

		for _, m := range group {
			line := " " + style.FormatUpdate(m.Path, m.Version, m.Update.Version, maxPathLen)
			if formats.Time {
				pt := format.PublishTime(m.Update.Time, deps.Now())
				if pt != "" {
					line += "  " + dim.Render(pt)
				}
			}
			fmt.Fprintln(deps.Out, line)
		}
	}

	printGroup("Direct dependencies (go.mod)", direct)
	printGroup("Indirect dependencies (go.mod // indirect)", indirect)
	if opts.All {
		printGroup("Transitive (not in go.mod)", transitive)
	}

	packagesToUpdate := make([]scanner.Module, 0, len(direct)+len(indirect)+len(transitive))
	packagesToUpdate = append(packagesToUpdate, direct...)
	packagesToUpdate = append(packagesToUpdate, indirect...)
	if opts.All {
		packagesToUpdate = append(packagesToUpdate, transitive...)
	}

	if opts.Upgrade {
		if deps.UpdatePackages == nil {
			return fmt.Errorf("missing deps.UpdatePackages")
		}
		fmt.Fprintln(deps.Out, "\nUpgrading...")
		if err := deps.UpdatePackages(packagesToUpdate); err != nil {
			return err
		}
		fmt.Fprintln(deps.Out, "Done.")
		return nil
	}

	fmt.Fprintln(deps.Out, "\nRun with -u to upgrade, or -i for interactive mode.")
	return nil
}
