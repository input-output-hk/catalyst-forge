package ci

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
)

const (
	RunStatusIdle    RunStatus = "idle"
	RunStatusRunning RunStatus = "running"
	RunStatusFailed  RunStatus = "failed"
	RunStatusSuccess RunStatus = "success"
)

var (
	checkMark     = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	ErrNoMoreRuns = errors.New("no more runs")
)

// RunStatus represents the status of a CI run.
type RunStatus string

type CIRunFinishedMsg struct {
	Run *CIRun
}

// CI represents a CI simulation.
type CI struct {
	filters  []string
	groups   []*CIRunGroup
	index    int
	loader   project.ProjectLoader
	logger   *slog.Logger
	options  []earthly.EarthlyExecutorOption
	scanPath string
}

// Finished returns true if the active run group has finished.
func (c *CI) Finished() bool {
	return c.groups[c.index].Finished()
}

// Load loads the CI runs to be executed.
func (c *CI) Load() error {
	w := walker.NewDefaultFSWalker(nil)
	var groups []*CIRunGroup

	projects, err := scan.ScanProjects(c.scanPath, c.loader, &w, c.logger)
	if err != nil {
		return err
	}

	for _, filter := range c.filters {
		var runs []*CIRun

		filterExpr, err := regexp.Compile(filter)
		if err != nil {
			return err
		}

		for _, project := range projects {
			if project.Earthfile != nil {
				targets := project.Earthfile.FilterTargets(func(target string) bool {
					return filterExpr.MatchString(target)
				})

				for _, target := range targets {
					runs = append(runs, &CIRun{
						Project: &project,
						Status:  RunStatusIdle,
						Target:  target,
						logger:  c.logger,
						options: c.options,
						spinner: spinner.New(),
					})
				}
			}
		}

		groups = append(groups, &CIRunGroup{
			Runs: runs,
		})
	}

	c.groups = groups
	return nil
}

// Next returns the next command to be executed. If there are no more runs, it
// returns an error.
func (c *CI) Next() (tea.Cmd, error) {
	c.index++
	if c.index >= len(c.groups) {
		return nil, ErrNoMoreRuns
	}

	return c.groups[c.index].Run(), nil
}

// Run starts the CI simulation.
func (c *CI) Run() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, c.groups[0].Run())
	for _, group := range c.groups {
		for _, run := range group.Runs {
			cmds = append(cmds, run.spinner.Tick)
		}
	}
	return tea.Batch(cmds...)
}

// UpdateSpinner updates the spinners of the CI simulation.
func (c *CI) UpdateSpinners(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	for _, group := range c.groups {
		for _, run := range group.Runs {
			cmds = append(cmds, run.UpdateSpinner(msg))
		}
	}

	return cmds
}

// View returns the current view of the CI simulation.
func (c *CI) View() string {
	if len(c.groups) > 0 && c.index < len(c.groups) {
		return c.groups[c.index].View()
	} else {
		return ""
	}
}

// CIRunGroup represents a group of CI runs.
type CIRunGroup struct {
	Runs []*CIRun
}

// Run starts the CI run group.
func (c *CIRunGroup) Run() tea.Cmd {
	var cmds []tea.Cmd
	for _, run := range c.Runs {
		cmds = append(cmds, run.Run)
	}

	return tea.Batch(cmds...)
}

// View returns the view of the CI run group.
func (c *CIRunGroup) View() string {
	var view string
	for _, run := range c.Runs {
		view += run.View() + "\n"
	}

	return strings.TrimSuffix(view, "\n")
}

// Finished returns true if all runs in the group have finished.
func (c *CIRunGroup) Finished() bool {
	for _, run := range c.Runs {
		if run.Status == RunStatusIdle || run.Status == RunStatusRunning {
			return false
		}
	}

	return true
}

// CIRun represents a CI run.
type CIRun struct {
	Project *project.Project
	Status  RunStatus
	Target  string
	logger  *slog.Logger
	options []earthly.EarthlyExecutorOption
	spinner spinner.Model
}

// Run starts the CI run.
func (c *CIRun) Run() tea.Msg {
	c.logger.Info("Running target", "project", c.Project.Path, "target", c.Target)
	c.Status = RunStatusRunning
	_, err := c.Project.RunTarget(
		c.Target,
		executor.NewLocalExecutor(c.logger),
		secrets.NewDefaultSecretStore(),
		c.options...,
	)

	if err != nil {
		c.logger.Error("Failed to run target", "project", c.Project.Path, "target", c.Target, "error", err)
		c.Status = RunStatusFailed
	} else {
		c.logger.Info("Target ran successfully", "project", c.Project.Path, "target", c.Target)
		c.Status = RunStatusSuccess
	}

	return CIRunFinishedMsg{
		Run: c,
	}
}

// UpdateSpinner updates the spinner of the CI run.
func (c *CIRun) UpdateSpinner(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if msg.ID == c.spinner.ID() {
			var cmd tea.Cmd
			c.spinner, cmd = c.spinner.Update(msg)
			return cmd
		}
	}

	return nil
}

// View returns the view of the CI run.
func (c *CIRun) View() string {
	switch c.Status {
	case RunStatusIdle:
		return fmt.Sprintf("%s %s", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).SetString("•"), c.Project.Path+"+"+c.Target)
	case RunStatusRunning:
		return fmt.Sprintf("%s %s", c.spinner.View(), c.Project.Path+"+"+c.Target)
	case RunStatusFailed:
		return fmt.Sprintf("%s %s", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).SetString("✗"), c.Project.Path+"+"+c.Target)
	case RunStatusSuccess:
		return fmt.Sprintf("%s %s", checkMark, c.Project.Path+"+"+c.Target)
	default:
		return ""
	}
}
