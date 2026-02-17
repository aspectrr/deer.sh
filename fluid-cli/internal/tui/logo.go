package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Version is the current version of Fluid (set via ldflags at build time)
var Version = "dev"

//nolint:staticcheck // ST1018: ANSI escape sequences are intentional for terminal colors
var BannerLogo = []string{
	"[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;40;43;50ms[0m[38;2;39;41;49mc[0m[38;2;40;44;51ms[0m[38;2;47;51;57m1[0m[38;2;51;56;62mx[0m[38;2;55;60;66m%[0m[38;2;61;66;72ma[0m[38;2;57;62;68m7[0m[38;2;51;55;62mx[0m[38;2;46;50;57mv[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m",
	"[38;2;41;44;51ms[0m[38;2;39;42;49mc[0m[38;2;41;44;51ms[0m[38;2;55;60;67m%[0m[38;2;93;103;108mC[0m[38;2;139;152;155m5[0m[38;2;167;183;185mO[0m[38;2;175;192;194mk[0m[38;2;178;195;197mP[0m[38;2;182;199;201mh[0m[38;2;183;199;201mh[0m[38;2;176;191;194mk[0m[38;2;119;132;137mL[0m[38;2;53;58;64mx[0m[38;2;42;45;52ms[0m[38;2;41;45;52ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m",
	"[38;2;46;50;57mv[0m[38;2;87;100;106mn[0m[38;2;128;145;149mq[0m[38;2;170;187;189mV[0m[38;2;200;218;219mA[0m[38;2;185;205;211mm[0m[38;2;158;181;195mF[0m[38;2;119;154;183mL[0m[38;2;125;161;190mp[0m[38;2;113;153;190m6[0m[38;2;128;160;183mq[0m[38;2;170;187;191mV[0m[38;2;157;171;174mF[0m[38;2;127;137;141mp[0m[38;2;92;102;107mu[0m[38;2;49;54;61m1[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m",
	"[38;2;65;72;79me[0m[38;2;168;192;197mV[0m[38;2;191;214;218mw[0m[38;2;198;217;221m8[0m[38;2;198;217;221m8[0m[38;2;193;213;218m4[0m[38;2;158;182;198mF[0m[38;2;138;168;195m2[0m[38;2;108;144;188m*[0m[38;2;70;118;184mj[0m[38;2;46;67;93mv[0m[38;2;53;57;63mx[0m[38;2;62;67;74ma[0m[38;2;60;65;72m7[0m[38;2;53;57;64mx[0m[38;2;42;45;52ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;41;44;51ms[0m[38;2;40;43;50ms[0m[38;2;39;42;49mc[0m[38;2;40;43;50ms[0m",
	"[38;2;55;63;73m%[0m[38;2;127;150;167mp[0m[38;2;92;122;154mu[0m[38;2;75;105;141mr[0m[38;2;84;117;154mf[0m[38;2;95;128;163mC[0m[38;2;73;110;157mo[0m[38;2;101;132;172mT[0m[38;2;154;177;204mg[0m[38;2;142;167;207m5[0m[38;2;98;120;154mJ[0m[38;2;62;70;84ma[0m[38;2;46;49;57mv[0m[38;2;40;43;50ms[0m[38;2;38;41;48mc[0m[38;2;39;41;48mc[0m[38;2;39;41;48mc[0m[38;2;39;42;48mc[0m[38;2;39;42;49mc[0m[38;2;41;43;50ms[0m[38;2;45;48;55mv[0m[38;2;54;59;66m%[0m[38;2;73;83;92mo[0m[38;2;69;79;91mj[0m",
	"[38;2;60;68;77m7[0m[38;2;145;168;181mS[0m[38;2;124;148;172mY[0m[38;2;96;121;153mJ[0m[38;2;113;138;166m6[0m[38;2;149;172;192mb[0m[38;2;171;191;205mX[0m[38;2;195;212;217m4[0m[38;2;208;225;225mD[0m[38;2;188;211;215mm[0m[38;2;168;193;205mV[0m[38;2;171;192;203mX[0m[38;2;155;172;179mg[0m[38;2;134;148;153my[0m[38;2;119;130;136mL[0m[38;2;104;116;122m3[0m[38;2;95;107;115mC[0m[38;2;97;109;116mJ[0m[38;2;101;114;122mT[0m[38;2;97;115;129mJ[0m[38;2;102;127;147mT[0m[38;2;103;137;168mT[0m[38;2;103;146;189mT[0m[38;2;71;99;135mj[0m",
	"[38;2;57;66;79m7[0m[38;2;102;135;179mT[0m[38;2;104;142;188m3[0m[38;2;130;164;192mq[0m[38;2;147;177;196mb[0m[38;2;179;204;211mP[0m[38;2;169;195;205mV[0m[38;2;180;204;211mP[0m[38;2;172;196;205mX[0m[38;2;154;180;193mg[0m[38;2;121;151;179mY[0m[38;2;109;140;172m*[0m[38;2;104;136;169m3[0m[38;2;108;141;173m*[0m[38;2;117;151;179m9[0m[38;2;110;147;176m*[0m[38;2;99;136;170mJ[0m[38;2;100;134;169mT[0m[38;2;104;138;171m3[0m[38;2;107;142;175m*[0m[38;2;110;146;177m*[0m[38;2;109;146;178m*[0m[38;2;113;153;185m6[0m[38;2;85;110;133mf[0m",
	"[38;2;41;49;65ms[0m[38;2;38;63;114mc[0m[38;2;42;77;130ms[0m[38;2;43;76;121mv[0m[38;2;52;82;120mx[0m[38;2;79;100;124mz[0m[38;2;72;96;122mo[0m[38;2;122;142;154mY[0m[38;2;138;155;163m2[0m[38;2;97;115;133mJ[0m[38;2;92;109;130mu[0m[38;2;89;106;128mu[0m[38;2;84;102;126mf[0m[38;2;87;107;132mn[0m[38;2;84;108;134mf[0m[38;2;78;104;133mr[0m[38;2;81;107;135mz[0m[38;2;89;111;135mu[0m[38;2;96;119;140mJ[0m[38;2;115;136;152m9[0m[38;2;120;142;157mL[0m[38;2;110;136;154m*[0m[38;2;107;133;152m*[0m[38;2;77;94;109mr[0m",
}

// GetBannerLogoWidth returns the display width of the logo (not counting ANSI codes)
func GetBannerLogoWidth() int {
	return 18 // Visual width including spacing
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

	// Get current working directory
	// cwd, err := os.Getwd()
	// if err != nil {
	// 	cwd = "~"
	// }
	// // Abbreviate home directory
	// if home, err := os.UserHomeDir(); err == nil && len(cwd) >= len(home) && cwd[:len(home)] == home {
	// 	cwd = "~" + cwd[len(home):]
	// }

	// // Truncate cwd if too long
	// maxCwdLen := width - GetBannerLogoWidth() - 10
	// if maxCwdLen < 20 {
	// 	maxCwdLen = 20
	// }
	// if len(cwd) > maxCwdLen {
	// 	cwd = "..." + cwd[len(cwd)-maxCwdLen+3:]
	// }

	// Build right-side info lines
	infoLines := []string{
		brandStyle.Render("Fluid.sh") + " " + versionStyle.Render("v"+Version),
		infoStyle.Render(modelName),
		cwdStyle.Render(hosts),
		// cwdStyle.Render(cwd),
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
