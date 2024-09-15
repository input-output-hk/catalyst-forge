package ci

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/forge/cli/tui"
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
			cmd, err := a.ci.Next()
			if err != nil {
				a.logger.Info("No more runs")
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

// Run starts the TUI application.
func Run(scanPath string,
	filters []string,
	local bool,
	opts ...earthly.EarthlyExecutorOption,
) error {
	logger, f, err := tui.NewLogger()
	if err != nil {
		return err
	}
	defer f.Close()

	loader := project.NewDefaultProjectLoader(
		false,
		local,
		project.GetDefaultRuntimes(logger),
		logger,
	)

	ci := CI{
		filters:  filters,
		loader:   &loader,
		logger:   logger,
		options:  opts,
		scanPath: scanPath,
	}

	logger.Info("Loading project")
	if err := ci.Load(); err != nil {
		return err
	}

	app := App{
		ci:     ci,
		logger: logger,
	}

	logger.Info("Starting program")
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
