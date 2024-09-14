package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

type run bool
type done bool

type model struct {
	log     *os.File
	height  int
	width   int
	spinner spinner.Model
	opts    []earthly.EarthlyExecutorOption
	project project.Project
	running bool
	target  string
}

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

func (m model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg { return run(true) }, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case done:
		fmt.Fprint(m.log, "Done\n")
		return m, tea.Sequence(
			tea.Printf("%s %s", checkMark, m.project.Path+"+"+m.target),
			tea.Quit,
		)
	case run:
		fmt.Fprint(m.log, "Running target\n")
		m.running = true
		return m, runTarget(m)
	}
	return m, nil
}

func (m model) View() string {
	var view string
	if m.running {
		spin := m.spinner.View() + " "
		path := m.project.Path + "+" + m.target
		view = spin + path
	}
	return view
}

func runTarget(m model) tea.Cmd {
	return func() tea.Msg {
		localExec := executor.NewLocalExecutor(
			testutils.NewNoopLogger(),
		)

		fmt.Fprintf(m.log, "Running target %s\n", m.target)
		m.project.RunTarget(
			m.target,
			localExec,
			secrets.NewDefaultSecretStore(),
			m.opts...,
		)

		return done(true)
	}
}

func Start(
	project project.Project,
	target string,
	opts ...earthly.EarthlyExecutorOption,
) error {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		return err
	}
	defer f.Close()

	model := model{
		log:     f,
		opts:    opts,
		project: project,
		spinner: spinner.New(),
		target:  target,
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
