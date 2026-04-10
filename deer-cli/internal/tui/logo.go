package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Version is the current version of Deer.sh (set via ldflags at build time)
var Version = "dev"

var BannerLogo = []string{
	"",
	"",
	"  🦌",
	"",
	"",
}

// GetBannerLogoWidth returns the display width of the logo (not counting ANSI codes)
func GetBannerLogoWidth() int {
	return 4 // Visual width: 2 spaces + emoji (2 cells)
}

// GetBannerLogoHeight returns the number of lines in the logo
func GetBannerLogoHeight() int {
	return len(BannerLogo)
}

// RenderBanner renders the startup banner with logo and info side by side
func RenderBanner(modelName string, hosts string, provider string, width int) string {
	// Styles
	brandStyle := lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	versionStyle := lipgloss.NewStyle().Foreground(mutedColor)
	infoStyle := lipgloss.NewStyle().Foreground(textColor)
	cwdStyle := lipgloss.NewStyle().Foreground(mutedColor)

	// Build right-side info lines
	infoLines := []string{
		brandStyle.Render("Deer.sh") + " " + versionStyle.Render("v"+Version),
		infoStyle.Render(modelName),
		cwdStyle.Render(hosts),
	}

	// Combine logo and info
	maxLines := len(BannerLogo)
	if len(infoLines) > maxLines {
		maxLines = len(infoLines)
	}

	// Calculate vertical offset to center info lines against logo
	infoOffset := (len(BannerLogo) - len(infoLines)) / 2
	if infoOffset < 0 {
		infoOffset = 0
	}

	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		var line string

		// Add logo line
		if i < len(BannerLogo) {
			line = BannerLogo[i]
		} else {
			line = strings.Repeat(" ", GetBannerLogoWidth())
		}

		// Add gap between logo and text
		line += "  "

		// Add info line (with offset for vertical centering)
		infoIdx := i - infoOffset
		if infoIdx >= 0 && infoIdx < len(infoLines) {
			line += infoLines[infoIdx]
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// RenderStatusBarBottom renders the bottom status bar with model, sandbox, mode, and context usage
func RenderStatusBarBottom(modelName string, sandboxID string, sandboxHost string, sandboxBaseImage string, sourceVM string, contextUsage float64, readOnly bool, width int) string {
	// Styles
	dividerStyle := lipgloss.NewStyle().Foreground(mutedColor)
	modelStyle := lipgloss.NewStyle().Foreground(textColor)
	sandboxStyle := lipgloss.NewStyle().Foreground(secondaryColor)
	sourceVMStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")) // Amber
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")) // Olive/green

	divider := dividerStyle.Render(" | ")

	// Model
	modelPart := modelStyle.Render(modelName)

	// Mode badge
	var modePart string
	if readOnly {
		modePart = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EAB308")).Render("READ-ONLY")
	} else {
		modePart = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981")).Render("EDIT")
	}

	// Target: source VM or sandbox
	var targetPart string
	if sourceVM != "" {
		// Currently operating on a source VM
		targetPart = sourceVMStyle.Render(sourceVM + " (read-only)")
	} else if sandboxID != "" {
		if sandboxBaseImage != "" {
			targetPart = sandboxStyle.Render(sandboxID + " (from " + sandboxBaseImage + ")")
		} else if sandboxHost != "" {
			targetPart = sandboxStyle.Render(sandboxID + " (" + sandboxHost + ")")
		} else {
			targetPart = sandboxStyle.Render(sandboxID)
		}
	} else {
		targetPart = dividerStyle.Render("no sandbox")
	}

	// Context bar
	barWidth := 10
	filled := int(contextUsage * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	bar := "[" + strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled) + "]"
	percentage := int(contextUsage * 100)
	contextPart := progressStyle.Render(bar) + " " + dividerStyle.Render(fmt.Sprintf("%d%%", percentage))

	// Combine all parts
	fullBar := modelPart + divider + modePart + divider + targetPart + divider + contextPart

	// Render with full width
	barStyle := lipgloss.NewStyle().
		Width(width).
		Foreground(mutedColor)

	return barStyle.Render(fullBar)
}
