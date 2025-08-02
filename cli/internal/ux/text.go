package ux

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Println prints a line of text with the default style.
func Println(s ...any) {
	fmt.Println(s...)
}

// Printfln prints a formatted line of text with the default style.
func Printfln(format string, a ...any) {
	fmt.Printf(format, a...)
	fmt.Println()
}

// Success prints a line of text with the success style.
func Success(s ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Success)
	fmt.Println(style.Render(fmt.Sprint(s...)))
}

// Successfln prints a formatted line of text with the success style.
func Successfln(format string, a ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Success)
	fmt.Println(style.Render(fmt.Sprintf(format, a...)))
}

// Warning prints a line of text with the warning style.
func Warning(s ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Warning)
	fmt.Println(style.Render(fmt.Sprint(s...)))
}

// Warningfln prints a formatted line of text with the warning style.
func Warningfln(format string, a ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Warning)
	fmt.Println(style.Render(fmt.Sprintf(format, a...)))
}

// Danger prints a line of text with the danger style.
func Danger(s ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Danger)
	fmt.Println(style.Render(fmt.Sprint(s...)))
}

// Dangerfln prints a formatted line of text with the danger style.
func Dangerfln(format string, a ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Danger)
	fmt.Println(style.Render(fmt.Sprintf(format, a...)))
}

// Info prints a line of text with the info style.
func Info(s ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Info)
	fmt.Println(style.Render(fmt.Sprint(s...)))
}

// Infofln prints a formatted line of text with the info style.
func Infofln(format string, a ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Info)
	fmt.Println(style.Render(fmt.Sprintf(format, a...)))
}

// Faint prints a line of text with the faint style.
func Faint(s ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Faint)
	fmt.Println(style.Render(fmt.Sprint(s...)))
}

// Faintfln prints a formatted line of text with the faint style.
func Faintfln(format string, a ...any) {
	style := lipgloss.NewStyle().Foreground(DefaultTheme.Faint)
	fmt.Println(style.Render(fmt.Sprintf(format, a...)))
}
