package pipeline

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
	"github.com/input-output-hk/catalyst-forge/forge/cli/tui"
)

type groupMsg struct {
	targets []*ProjectTarget
}

type finishedMsg struct {
	index int
}

type model struct {
	log        *os.File
	height     int
	width      int
	spinner    spinner.Model
	opts       []earthly.EarthlyExecutorOption
	groupIndex int
	runs       [][]*ProjectTarget
}

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

func (m model) Init() tea.Cmd {
	fmt.Fprint(m.log, "Starting pipeline\n")
	return tea.Batch(runGroup(m), m.spinner.Tick)
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
	case groupMsg:
		fmt.Fprintf(m.log, "Running group %d\n", m.groupIndex)
		var cmds []tea.Cmd
		for i := range msg.targets {
			m.runs[m.groupIndex][i].status = runStatusRunning
			cmds = append(cmds, runTarget(m, i))
		}

		return m, tea.Batch(cmds...)
	case finishedMsg:
		fmt.Fprintf(m.log, "Got finished message for target %d\n", msg.index)
		fmt.Fprintf(m.log, "Pairs: %+v\n", m.runs[m.groupIndex])
		for _, pair := range m.runs[m.groupIndex] {
			if pair.status == runStatusRunning {
				return m, nil
			}
		}

		m.groupIndex++
		if m.groupIndex < len(m.runs) {
			fmt.Fprintf(m.log, "Group %d finished\n", m.groupIndex-1)

			return m, tea.Sequence(
				tea.Println(makeStatusView(m.runs[m.groupIndex-1], "")),
				runGroup(m),
			)
		} else {
			return m, tea.Sequence(
				tea.Println(makeStatusView(m.runs[m.groupIndex-1], "")),
				tea.Quit,
			)
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.groupIndex < len(m.runs) {
		return makeStatusView(m.runs[m.groupIndex], m.spinner.View())
	} else {
		return ""
	}
}

func makeStatusView(runs []*ProjectTarget, spin string) string {
	var view string

	for _, pair := range runs {
		switch pair.status {
		case runStatusIdle:
			view += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).SetString("•"), pair.project.Path+"+"+pair.target)
		case runStatusRunning:
			view += fmt.Sprintf("%s %s\n", spin, pair.project.Path+"+"+pair.target)
		case runStatusFailed:
			view += fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).SetString("✗"), pair.project.Path+"+"+pair.target)
		case runStatusSuccess:
			view += fmt.Sprintf("%s %s\n", checkMark, pair.project.Path+"+"+pair.target)
		}
	}

	return strings.TrimSuffix(view, "\n")
}

func runGroup(m model) tea.Cmd {
	return func() tea.Msg {
		return groupMsg{targets: m.runs[m.groupIndex]}
	}
}

func runTarget(m model, index int) tea.Cmd {
	return func() tea.Msg {
		pair := m.runs[m.groupIndex][index]
		localExec := executor.NewLocalExecutor(
			testutils.NewNoopLogger(),
		)

		fmt.Fprintf(m.log, "Running target %s+%s\n", pair.project.Path, pair.target)
		_, err := pair.project.RunTarget(
			pair.target,
			localExec,
			secrets.NewDefaultSecretStore(),
			m.opts...,
		)

		if err != nil {
			fmt.Fprintf(m.log, "Failed to run target %s+%s: %s\n", pair.project.Path, pair.target, err)
			pair.status = runStatusFailed
		} else {
			fmt.Fprintf(m.log, "Successfully ran target %s+%s\n", pair.project.Path, pair.target)
			pair.status = runStatusSuccess
		}

		return finishedMsg{
			index: index,
		}
	}
}

func Start(
	startPath string,
	filters []string,
	local bool,
	opts ...earthly.EarthlyExecutorOption,
) error {
	log, err := tui.MakeDebugFile()
	if err != nil {
		return err
	}
	defer log.Close()

	loader := project.NewDefaultProjectLoader(
		false,
		local,
		project.GetDefaultRuntimes(testutils.NewNoopLogger()),
		testutils.NewNoopLogger(),
	)

	fmt.Fprintf(log, "Scanning projects in %s\n", startPath)
	runs, err := scanProjects(startPath, &loader, filters)
	if err != nil {
		return err
	}

	model := model{
		log:     log,
		opts:    opts,
		runs:    runs,
		spinner: spinner.New(),
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
