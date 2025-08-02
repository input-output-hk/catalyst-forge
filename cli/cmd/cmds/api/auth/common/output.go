package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

// User output functions
func OutputUserJSON(user *client.User) error {
	jsonData, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func OutputUserTable(user *client.User) error {
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
		Headers("ID", "Email", "Status", "Created At").
		Rows(
			[]string{
				fmt.Sprintf("%d", user.ID),
				user.Email,
				user.Status,
				user.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		)

	fmt.Println(t)
	return nil
}

func OutputUsersJSON(users []client.User) error {
	jsonData, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func OutputUsersTable(users []client.User) error {
	if len(users) == 0 {
		fmt.Println("No users found.")
		return nil
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
		Headers("ID", "Email", "Status", "Created At")

	var rows [][]string
	for _, user := range users {
		rows = append(rows, []string{
			fmt.Sprintf("%d", user.ID),
			user.Email,
			user.Status,
			user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	t = t.Rows(rows...)
	fmt.Println(t)
	return nil
}

// UserKey output functions
func OutputUserKeyJSON(userKey *client.UserKey) error {
	jsonData, err := json.MarshalIndent(userKey, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func OutputUserKeyTable(userKey *client.UserKey) error {
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
		Headers("ID", "User ID", "KID", "Status", "Created At").
		Rows(
			[]string{
				fmt.Sprintf("%d", userKey.ID),
				fmt.Sprintf("%d", userKey.UserID),
				userKey.Kid,
				userKey.Status,
				userKey.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		)

	fmt.Println(t)
	return nil
}

func OutputUserKeysJSON(userKeys []client.UserKey) error {
	jsonData, err := json.MarshalIndent(userKeys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func OutputUserKeysTable(userKeys []client.UserKey) error {
	if len(userKeys) == 0 {
		fmt.Println("No user keys found.")
		return nil
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
		Headers("ID", "User ID", "KID", "Status", "Created At")

	var rows [][]string
	for _, userKey := range userKeys {
		rows = append(rows, []string{
			fmt.Sprintf("%d", userKey.ID),
			fmt.Sprintf("%d", userKey.UserID),
			userKey.Kid,
			userKey.Status,
			userKey.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	t = t.Rows(rows...)
	fmt.Println(t)
	return nil
}

// Role output functions
func OutputRoleJSON(role *client.Role) error {
	jsonData, err := json.MarshalIndent(role, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func OutputRoleTable(role *client.Role) error {
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
		Headers("ID", "Name", "Description", "Permissions", "Created At").
		Rows(
			[]string{
				fmt.Sprintf("%d", role.ID),
				role.Name,
				role.Description,
				strings.Join(role.Permissions, "\n"),
				role.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		)

	fmt.Println(t)
	return nil
}

func OutputRolesJSON(roles []client.Role) error {
	jsonData, err := json.MarshalIndent(roles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func OutputRolesTable(roles []client.Role) error {
	if len(roles) == 0 {
		fmt.Println("No roles found.")
		return nil
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
		Headers("ID", "Name", "Description", "Permissions", "Created At")

	var rows [][]string
	for _, role := range roles {
		rows = append(rows, []string{
			fmt.Sprintf("%d", role.ID),
			role.Name,
			role.Description,
			strings.Join(role.Permissions, "\n"),
			role.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	t = t.Rows(rows...)
	fmt.Println(t)
	return nil
}
