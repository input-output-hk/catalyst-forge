package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/github"
)

func outputJSON(auth *github.GithubRepositoryAuth) error {
	jsonData, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func outputTable(auth *github.GithubRepositoryAuth) error {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("62"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
			case row%2 == 0:
				return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			default:
				return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
			}
		}).
		Headers("ID", "Repository", "Enabled", "Description", "Permissions").
		Rows(
			[]string{
				fmt.Sprintf("%d", auth.ID),
				auth.Repository,
				fmt.Sprintf("%t", auth.Enabled),
				auth.Description,
				strings.Join(auth.Permissions, "\n"),
			},
		)

	fmt.Println(t)
	return nil
}
