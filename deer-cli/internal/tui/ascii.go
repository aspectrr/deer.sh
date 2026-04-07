package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FluidLogo returns the FLUID ASCII art logo in blue
func FluidLogo() string {
	logo := `
 ███████╗██╗     ██╗   ██╗██╗██████╗
 ██╔════╝██║     ██║   ██║██║██╔══██╗
 █████╗  ██║     ██║   ██║██║██║  ██║
 ██╔══╝  ██║     ██║   ██║██║██║  ██║
 ██║     ███████╗╚██████╔╝██║██████╔╝
 ╚═╝     ╚══════╝ ╚═════╝ ╚═╝╚═════╝
`
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true)

	return style.Render(strings.TrimPrefix(logo, "\n"))
}

// FluidLogoSmall returns a smaller version of the logo
func FluidLogoSmall() string {
	logo := `
 ╔═╗╦  ╦ ╦╦╔╦╗
 ╠╣ ║  ║ ║║ ║║
 ╚  ╩═╝╚═╝╩═╩╝
`
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true)

	return style.Render(strings.TrimPrefix(logo, "\n"))
}

// BoxedText renders text in a nice box
func BoxedText(title, content string, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B82F6")).
		Padding(1, 2).
		Width(width)

	var b strings.Builder
	if title != "" {
		b.WriteString(titleStyle.Render(title))
		b.WriteString("\n\n")
	}
	b.WriteString(content)

	return boxStyle.Render(b.String())
}
