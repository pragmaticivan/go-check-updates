// Package tui provides the interactive terminal user interface for selecting updates.
package tui

import (
	"fmt"
	"os"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pragmaticivan/go-check-updates/internal/format"
	"github.com/pragmaticivan/go-check-updates/internal/scanner"
	"github.com/pragmaticivan/go-check-updates/internal/style"
	"github.com/pragmaticivan/go-check-updates/internal/updater"
)

var runProgram = func(m tea.Model) (tea.Model, error) {
	p := tea.NewProgram(m)
	return p.Run()
}

var updatePackages = updater.UpdatePackages

// Options configures rendering and grouping behavior for the interactive TUI.
type Options struct {
	FormatGroup bool
	FormatTime  bool
}

type model struct {
	choices  []scanner.Module
	selected map[int]struct{}
	cursor   int
	quitting bool

	directEnd    int
	indirectEnd  int
	transitiveOn bool

	opts Options
}

func initialModel(direct, indirect, transitive []scanner.Module, opts Options) model {
	if opts.FormatGroup {
		sort.Slice(direct, func(i, j int) bool {
			ai, aj := format.GroupSortKey(direct[i]), format.GroupSortKey(direct[j])
			if ai != aj {
				return ai < aj
			}
			return direct[i].Path < direct[j].Path
		})
		sort.Slice(indirect, func(i, j int) bool {
			ai, aj := format.GroupSortKey(indirect[i]), format.GroupSortKey(indirect[j])
			if ai != aj {
				return ai < aj
			}
			return indirect[i].Path < indirect[j].Path
		})
		sort.Slice(transitive, func(i, j int) bool {
			ai, aj := format.GroupSortKey(transitive[i]), format.GroupSortKey(transitive[j])
			if ai != aj {
				return ai < aj
			}
			return transitive[i].Path < transitive[j].Path
		})
	}

	choices := make([]scanner.Module, 0, len(direct)+len(indirect)+len(transitive))
	choices = append(choices, direct...)
	directEnd := len(choices)
	choices = append(choices, indirect...)
	indirectEnd := len(choices)
	choices = append(choices, transitive...)

	return model{
		choices:      choices,
		selected:     make(map[int]struct{}),
		directEnd:    directEnd,
		indirectEnd:  indirectEnd,
		transitiveOn: len(transitive) > 0,
		opts:         opts,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case " ", "space":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	headingMuted := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))

	s := "Which packages would you like to update?\n\n"

	// Find longest path for padding
	maxPathLen := 0
	for _, c := range m.choices {
		if len(c.Path) > maxPathLen {
			maxPathLen = len(c.Path)
		}
	}

	prevGroup := ""
	for i, choice := range m.choices {
		// Section headings (do not affect cursor/selection indices)
		if i == 0 {
			s += heading.Render("Direct dependencies (go.mod)") + "\n"
			prevGroup = ""
		}
		if i == m.directEnd && i < len(m.choices) {
			s += "\n" + headingMuted.Render("Indirect dependencies (go.mod // indirect)") + "\n"
			prevGroup = ""
		}
		if m.transitiveOn && i == m.indirectEnd && i < len(m.choices) {
			s += "\n" + headingMuted.Render("Transitive (not in go.mod)") + "\n"
			prevGroup = ""
		}

		if m.opts.FormatGroup {
			g := format.GroupLabel(choice)
			if g != prevGroup {
				s += "\n" + dim.Render(g) + "\n"
				prevGroup = g
			}
		}

		// Cursor
		cursor := "  "
		if m.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("❯ ")
		}

		// Checkbox
		var checked string
		if _, ok := m.selected[i]; ok {
			checked = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("◉")
		} else {
			checked = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("◯")
		}

		// Row content
		row := style.FormatUpdate(choice.Path, choice.Version, choice.Update.Version, maxPathLen)
		if m.opts.FormatTime && choice.Update != nil {
			pt := format.PublishTime(choice.Update.Time, time.Now())
			if pt != "" {
				row += "  " + dim.Render(pt)
			}
		}

		s += fmt.Sprintf("%s%s %s\n", cursor, checked, row)
	}

	s += "\nPress <space> to select, <enter> to update, <q> to quit.\n"
	return s
}

// StartInteractiveGroupedWithOptions launches the TUI with groups split by go.mod classification.
func StartInteractiveGroupedWithOptions(direct, indirect, transitive []scanner.Module, opts Options) {
	m, err := runProgram(initialModel(direct, indirect, transitive, opts))
	if err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}

	// Type assertion to get back our model
	if finalModel, ok := m.(model); ok && !finalModel.quitting {
		// Collect selected modules
		var toUpdate []scanner.Module
		for i := range finalModel.selected {
			toUpdate = append(toUpdate, finalModel.choices[i])
		}

		if len(toUpdate) > 0 {
			if err := updatePackages(toUpdate); err != nil {
				fmt.Printf("Error updating: %v\n", err)
			} else {
				fmt.Println("Updates complete!")
			}
		} else {
			fmt.Println("No packages selected.")
		}
	}
}

// StartInteractiveGrouped is a backwards-compatible helper.
func StartInteractiveGrouped(direct, indirect, transitive []scanner.Module) {
	StartInteractiveGroupedWithOptions(direct, indirect, transitive, Options{})
}
