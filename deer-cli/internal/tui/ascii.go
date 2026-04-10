package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DeerLogo returns the DEER ASCII art logo in forest green
func DeerLogo() string {
	logo := `
 ██████╗ ███████╗███████╗██████╗
 ██╔══██╗██╔════╝██╔════╝██╔══██╗
 ██║  ██║█████╗  █████╗  ██████╔╝
 ██║  ██║██╔══╝  ██╔══╝  ██╔══██╗
 ██████╔╝███████╗███████╗██║  ██║
 ╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝
`
	style := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	return style.Render(strings.TrimPrefix(logo, "\n"))
}

// DeerLogoSmall returns a smaller version of the logo
func DeerLogoSmall() string {
	logo := `
 ██████╗
 ██╔══██╗
 ██║  ██║
 ╚═════╝
`
	style := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	return style.Render(strings.TrimPrefix(logo, "\n"))
}

// BoxedText renders text in a nice box
func BoxedText(title, content string, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
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
