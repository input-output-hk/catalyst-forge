package ux

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
)

// NewForm creates a new form with the default theme.
func NewForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(huh.ThemeCharm())
}

// NewSpinner creates a new spinner with the default theme.
func NewSpinner() *spinner.Spinner {
	return spinner.New().Type(spinner.Line)
}
