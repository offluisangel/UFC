package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorRed   = lipgloss.Color("#E10600")
	colorWhite = lipgloss.Color("#FFFFFF")
	colorGray  = lipgloss.Color("#767676")
	colorDark  = lipgloss.Color("#1A1A1A")
	colorLine  = lipgloss.Color("#2E2E2E")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleAccent = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorGray)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorGray)

	styleValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorRed).
			Padding(0, 2)

	styleTabIdle = lipgloss.NewStyle().
			Foreground(colorGray).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorLine).
			Padding(0, 2)

	styleCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	styleHelpKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	styleHelpTxt = lipgloss.NewStyle().
			Foreground(colorGray)

	styleMainEventCard = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorRed).
				Padding(1, 2)

	styleMainEventLabel = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorRed)

	styleMainEventName = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite)

	styleMainEventVS = lipgloss.NewStyle().
				Foreground(colorGray)
)

func panelStyle(width int, active bool) lipgloss.Style {
	border := colorLine
	if active {
		border = colorRed
	}
	return lipgloss.NewStyle().
		Width(max(1, width-6)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Padding(1, 2)
}

func panelStyleH(width, height int, active bool) lipgloss.Style {
	border := colorLine
	if active {
		border = colorRed
	}
	w := max(1, width-6)
	s := lipgloss.NewStyle().
		Width(w).
		MaxWidth(max(1, width-2)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Padding(1, 2)
	if height > 4 {
		contentH := height - 4
		s = s.Height(contentH).MaxHeight(contentH)
	}
	return s
}

func panelHeading(label string, right string) string {
	left := styleSection.Render(label)
	if right == "" {
		return left
	}
	return left + styleMuted.Render("  "+right)
}

func panelHeadingW(label string, right string, width int) string {
	left := styleSection.Render(label)
	if right == "" {
		return left
	}
	r := styleMuted.Render(right)
	space := max(1, width-lipgloss.Width(left)-lipgloss.Width(r))
	return left + strings.Repeat(" ", space) + r
}

func fightLine(fighters []string) string {
	if len(fighters) == 0 {
		return styleMuted.Render("cartelera no anunciada")
	}

	var left, right string
	if len(fighters) > 0 {
		left = fighters[0]
	}
	if len(fighters) > 1 {
		right = fighters[1]
	}

	return styleValue.Render(left) + "  " +
		styleMainEventVS.Render("vs") + "  " +
		styleValue.Render(right)
}

func mainEventBox(fighters []string, weight string) string {
	var left, right string
	if len(fighters) > 0 {
		left = fighters[0]
	}
	if len(fighters) > 1 {
		right = fighters[1]
	}

	header := styleMainEventLabel.Render("☆  PELEA ESTELAR")
	names := styleMainEventName.Render(left) + "  " +
		styleMainEventVS.Render("vs") + "  " +
		styleMainEventName.Render(right)

	content := []string{header, "", names}
	if weight != "" {
		content = append(content, styleMuted.Render(weight))
	}

	return styleMainEventCard.Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}
