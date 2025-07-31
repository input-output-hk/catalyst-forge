package gha

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

type ListCmd struct {
	JSON bool `short:"j" help:"Output as prettified JSON instead of table."`
}

func (c *ListCmd) Run(ctx run.RunContext, cl client.Client) error {
	auths, err := cl.ListAuths(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list authentication entries: %w", err)
	}

	if c.JSON {
		return outputJSONList(auths)
	}

	return outputTableList(auths)
}

func outputJSONList(auths []client.GHARepositoryAuth) error {
	jsonData, err := json.MarshalIndent(auths, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func outputTableList(auths []client.GHARepositoryAuth) error {
	if len(auths) == 0 {
		fmt.Println("No authentication entries found.")
		return nil
	}

	var rows [][]string
	for _, auth := range auths {
		// Truncate permissions if too long, show count if many
		permissions := auth.Permissions
		if len(permissions) > 3 {
			permissions = permissions[:3]
		}
		permissionsStr := strings.Join(permissions, ", ")
		if len(auth.Permissions) > 3 {
			permissionsStr += fmt.Sprintf(" (+%d more)", len(auth.Permissions)-3)
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", auth.ID),
			auth.Repository,
			fmt.Sprintf("%t", auth.Enabled),
			auth.Description,
			permissionsStr,
		})
	}

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
		Rows(rows...).
		Width(120)

	fmt.Println(t)
	return nil
}
