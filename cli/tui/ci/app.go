package ci

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/tui"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

var (
	errStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
)

// App represents the TUI application.
type App struct {
	ci     CI
	logger *slog.Logger
	window tui.Window
}

func (a App) Init() tea.Cmd {
	a.logger.Info("Starting CI simulation")
	return a.ci.Run()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.window.Resize(msg)
		return a, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return a, tea.Quit
		}
	case spinner.TickMsg:
		return a, tea.Batch(a.ci.UpdateSpinners(msg)...)
	case CIRunFinishedMsg:
		a.logger.Info("Received CI run finished message", "project", msg.Run.Project.Path, "target", msg.Run.Target)
		if a.ci.Finished() {
			a.logger.Info("All CI runs finished for current group")
			out := a.ci.View()

			if failed := a.ci.Failed(); failed != nil {
				a.logger.Info("Group failed")
				a.ci.Stop()

				out += strings.Trim(errStyle.Render("\n\nRun failed, dumping logs\n\n"), " ")
				for _, run := range failed {
					out += errStyle.Render(
						fmt.Sprintf("%s+%s\n%s", run.Project.Path, run.Target, a.line()),
					)
					out += run.Stderr() + "\n\n"
				}

				return a, tea.Sequence(
					tea.Println(out),
					tea.Quit,
				)
			}

			cmd, err := a.ci.Next()
			if err != nil {
				a.logger.Info("No more runs")
				a.ci.Stop()

				out += strings.Trim(successStyle.Render("\n\nAll runs succeeded"), " ")
				return a, tea.Sequence(
					tea.Println(out),
					tea.Quit,
				)
			}

			a.logger.Info("Starting next group")
			return a, tea.Sequence(
				tea.Println(out),
				cmd,
			)
		}
	}

	return a, nil
}

func (a App) View() string {
	return a.ci.View()
}

// line returns a line of dashes the width of the window.
func (a App) line() string {
	return strings.Repeat("-", a.window.Width)
}

// Run starts the TUI application.
func Run(scanPath string,
	runctx run.RunContext,
	opts ...earthly.EarthlyExecutorOption,
) error {
	var logger *slog.Logger
	if _, ok := os.LookupEnv("DEBUG"); ok {
		var err error
		var f *os.File

		logger, f, err = tui.NewLogger()
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	loader := project.NewDefaultProjectLoader(
		project.GetDefaultRuntimes(logger),
		logger,
	)

	if scanPath == "" {
		scanPath, err := findRoot(".", logger)
		if err != nil {
			return fmt.Errorf("failed to find root of git repository: %w", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		scanPath, err = filepath.Rel(cwd, scanPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if scanPath == "" {
			scanPath = "."
		}
	}

	project, err := loader.Load(scanPath)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}
	if len(project.Blueprint.Global.CI.Local) <= 0 {
		return fmt.Errorf("no local CI filters found in project")
	}

	ci := CI{
		filters:  project.Blueprint.Global.CI.Local,
		loader:   &loader,
		logger:   logger,
		options:  opts,
		scanPath: scanPath,
	}

	logger.Info("Loading CI")
	if err := ci.Load(); err != nil {
		return fmt.Errorf("failed to load CI: %w", err)
	}

	app := App{
		ci:     ci,
		logger: logger,
	}

	logger.Info("Starting program")
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	return nil
}

// findRoot finds the root of a git repository.
func findRoot(scanPath string, logger *slog.Logger) (string, error) {
	rw := walker.NewDefaultFSReverseWalker(logger)
	return git.FindGitRoot(scanPath, &rw)
}
