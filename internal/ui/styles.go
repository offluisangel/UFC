package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/offluisangel/UFC/internal/model"
)

var (
	colorRed   = lipgloss.Color("#E10600")
	colorWhite = lipgloss.Color("#F2F2F2")
	colorGray  = lipgloss.Color("#8C8C8C")
	colorDim   = lipgloss.Color("#555555")
	colorLine  = lipgloss.Color("#333333")
	colorSel   = lipgloss.Color("#2B1210")
	colorGold  = lipgloss.Color("#F5B700")

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

	styleDim = lipgloss.NewStyle().
			Foreground(colorDim)

	styleRule = lipgloss.NewStyle().
			Foreground(colorLine)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorGray)

	styleValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleStar = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGold)

	styleBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorRed).
			Padding(0, 1)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorRed)

	styleTabNum = lipgloss.NewStyle().
			Foreground(colorRed)

	styleTabIdle = lipgloss.NewStyle().
			Foreground(colorGray)

	styleCursor = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	styleHelpKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleHelpTxt = lipgloss.NewStyle().
			Foreground(colorGray)

	styleMainEventBar = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder(), false, false, false, true).
				BorderForeground(colorRed).
				PaddingLeft(1)

	styleMainEventLabel = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorRed)

	styleMainEventName = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite)

	styleMainEventVS = lipgloss.NewStyle().
				Foreground(colorGray)
)

func panelStyleH(width, height int, active bool) lipgloss.Style {
	border := colorLine
	if active {
		border = colorRed
	}
	s := lipgloss.NewStyle().
		Width(max(1, width-2)).
		MaxWidth(max(1, width)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2)
	if height > 4 {
		s = s.Height(height - 2).MaxHeight(height)
	}
	return s
}

func rule(width int) string {
	return styleRule.Render(repeat("─", max(1, width)))
}

func headerLine(label, right string, width int) string {
	left := styleAccent.Render("▍") + " " + styleSection.Render(label)
	line := left
	if right != "" {
		r := styleMuted.Render(right)
		if gap := width - lipgloss.Width(left) - lipgloss.Width(r); gap >= 2 {
			line = left + strings.Repeat(" ", gap) + r
		}
	}
	return lipgloss.NewStyle().MaxWidth(max(1, width)).Render(line)
}

func sectionHeader(label, right string, width int) string {
	return headerLine(label, right, width) + "\n" + rule(width)
}

func cell(st lipgloss.Style, text string, selected bool) string {
	if selected {
		st = st.Background(colorSel)
	}
	return st.Render(text)
}

func spaces(n int, selected bool) string {
	return cell(lipgloss.NewStyle(), repeat(" ", n), selected)
}

func fillRow(line string, width int, selected bool) string {
	line = lipgloss.NewStyle().MaxWidth(max(1, width)).Render(line)
	if gap := width - lipgloss.Width(line); gap > 0 {
		line += spaces(gap, selected)
	}
	return line
}

func fighterNames(fighters []string) (string, string) {
	var left, right string
	if len(fighters) > 0 {
		left = fighters[0]
	}
	if len(fighters) > 1 {
		right = fighters[1]
	}
	return left, right
}

func mainEventBox(fighters []string, weight string) string {
	left, right := fighterNames(fighters)

	header := styleStar.Render("☆") + " " + styleMainEventLabel.Render("PELEA ESTELAR")
	names := styleMainEventName.Render(left) + "  " +
		styleMainEventVS.Render("vs") + "  " +
		styleMainEventName.Render(right)

	content := []string{header, "", names}
	if weight != "" {
		content = append(content, styleMuted.Render(weight))
	}
	return styleMainEventBar.Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

func fightOutcome(f model.Fight) string {
	if f.Method == "" {
		return ""
	}
	out := f.Method
	if f.Round != "" {
		out += " R" + f.Round
	}
	if f.Time != "" {
		out += " " + f.Time
	}
	return out
}
