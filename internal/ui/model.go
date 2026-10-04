package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	viewNext view = iota
	viewTonight
	viewCalendar
	viewCard
	viewResults
)

const numViews = 5

var viewNames = []string{"próximo", "hoy", "calendario", "cartelera", "resultados"}

type Model struct {
	dataset model.Dataset
	current view
	cursor  int
	width   int
	height  int
	ready   bool
	err     error
}

func New(ds model.Dataset, err error) Model {
	return Model{dataset: ds, err: err}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "1":
			m.current = viewNext
		case "2":
			m.current = viewTonight
		case "3":
			m.current = viewCalendar
		case "4":
			m.current = viewCard
		case "5":
			m.current = viewResults
		case "right", "l", "tab":
			m.current = (m.current + 1) % numViews
		case "left", "h", "shift+tab":
			m.current = (m.current + numViews - 1) % numViews
		case "up", "k":
			m.cursor--
		case "down", "j":
			m.cursor++
		}
		m.cursor = clampCursor(m.cursor, m.listLen())
	}
	return m, nil
}

func (m Model) listLen() int {
	switch m.current {
	case viewNext:
		e, ok := dataset.Next(m.dataset)
		if !ok {
			return 0
		}
		return len(e.Card)
	case viewCard:
		e, ok := dataset.Next(m.dataset)
		if !ok {
			return 0
		}
		return len(e.Card)
	case viewResults:
		return len(dataset.RecentResults(m.dataset, resultsLimit))
	default:
		return 0
	}
}

func clampCursor(cursor, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= n {
		return n - 1
	}
	return cursor
}

func (m Model) View() string {
	if !m.ready {
		return "cargando..."
	}
	if m.err != nil {
		return errorView(m.err)
	}

	width := m.width
	if width < 20 {
		width = 20
	}

	header := m.headerView(width)
	headerHeight := lipgloss.Height(header)
	help := m.helpView()
	helpHeight := lipgloss.Height(help)

	bodyHeight := m.height - headerHeight - helpHeight - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	switch m.current {
	case viewNext:
		body = m.nextView(width)
	case viewTonight:
		body = m.tonightView(width)
	case viewCalendar:
		body = m.calendarView(width)
	case viewCard:
		body = m.cardView(width)
	case viewResults:
		body = m.resultsView(width)
	}

	if m.height >= minCardHeight {
		body = clampLines(body, bodyHeight)
	}
	body = lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Height(bodyHeight).
		Render(body)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		body,
		"",
		help,
	)
}

func clampLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

func (m Model) headerView(width int) string {
	title := styleAccent.Render("UFC") + " " + styleTitle.Render("fight nights")
	count := styleMuted.Render(fmt.Sprintf("[ %d eventos ]", len(dataset.Upcoming(m.dataset))))
	rightW := max(0, width-lipgloss.Width(title))
	var rightStr string
	if rightW >= lipgloss.Width(count) {
		rightStr = lipgloss.NewStyle().Width(rightW).Align(lipgloss.Right).Render(count)
	}
	titleRow := lipgloss.NewStyle().MaxWidth(width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, title, rightStr),
	)

	tabs := make([]string, 0, numViews)
	for i, n := range viewNames {
		if view(i) == m.current {
			tabs = append(tabs, styleTabActive.Render(n))
		} else {
			tabs = append(tabs, styleTabIdle.Render(n))
		}
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	var tabBar string
	if width > lipgloss.Width(tabsRow) {
		remainingWidth := width - lipgloss.Width(tabsRow)
		fillRule := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorLine).
			Render(repeat(" ", remainingWidth))
		tabBar = lipgloss.JoinHorizontal(lipgloss.Top, tabsRow, fillRule)
	} else {
		tabBar = tabsRow
	}
	tabBar = lipgloss.NewStyle().MaxWidth(width).Render(tabBar)

	return lipgloss.NewStyle().MaxWidth(width).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			titleRow,
			"",
			tabBar,
		),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) helpView() string {
	pipeSepWide := lipgloss.NewStyle().Foreground(colorLine).Render("  |  ")
	pipeSepTight := lipgloss.NewStyle().Foreground(colorLine).Render(" | ")

	items := make([]string, 0, numViews)
	for i, n := range viewNames {
		items = append(items, styleHelpKey.Render(strconv.Itoa(i+1))+styleHelpTxt.Render(" "+n))
	}

	navKey := "←/→"
	navTxt := "cambiar"
	if m.current == viewCard {
		navKey = "↑/↓"
		navTxt = "peleas"
	} else if m.current == viewResults {
		navKey = "↑/↓"
		navTxt = "eventos"
	}

	nav := styleHelpKey.Render(navKey) + styleHelpTxt.Render(" "+navTxt)
	quit := styleHelpKey.Render("q") + styleHelpTxt.Render(" salir")

	allParts := append(append([]string{}, items...), nav, quit)
	fullWithPipes := strings.Join(allParts, pipeSepWide)
	if lipgloss.Width(fullWithPipes) <= m.helpWidth() {
		return fullWithPipes
	}

	fullWithPipesTight := strings.Join(allParts, pipeSepTight)
	if lipgloss.Width(fullWithPipesTight) <= m.helpWidth() {
		return fullWithPipesTight
	}

	tabList2 := strings.Join(items, "  ")
	fullGrouped := tabList2 + pipeSepTight + nav + pipeSepTight + quit
	if lipgloss.Width(fullGrouped) <= m.helpWidth() {
		return fullGrouped
	}

	tabList1 := strings.Join(items, " ")
	fullCompact := tabList1 + pipeSepTight + nav + pipeSepTight + quit
	if lipgloss.Width(fullCompact) <= m.helpWidth() {
		return fullCompact
	}

	tabListQuit := tabList2 + "  " + quit
	if lipgloss.Width(tabListQuit) <= m.helpWidth() {
		return tabListQuit
	}
	tabListQuitCompact := tabList1 + "  " + quit
	if lipgloss.Width(tabListQuitCompact) <= m.helpWidth() {
		return tabListQuitCompact
	}

	currTab := styleHelpKey.Render(strconv.Itoa(int(m.current)+1)) + styleHelpTxt.Render(" "+viewNames[m.current])
	currLine := currTab + pipeSepTight + nav + pipeSepTight + quit
	if lipgloss.Width(currLine) <= m.helpWidth() {
		return currLine
	}

	currLine2 := currTab + "  " + quit
	if lipgloss.Width(currLine2) <= m.helpWidth() {
		return currLine2
	}

	return quit
}

func (m Model) helpWidth() int {
	if m.width < 20 {
		return 20
	}
	return m.width
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

func field(label, value string) string {
	l := styleLabel.Render(pad(label, 12))
	v := styleValue.Render(value)
	return l + v
}

func pad(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + repeat(" ", n-w)
}

func daysUntil(target time.Time) int {
	t := dataset.DateOnly(target)
	return int(t.Sub(dataset.Today()).Hours() / 24)
}

func announce(n int) string {
	if n == 1 {
		return "1 pelea"
	}
	return fmt.Sprintf("%d peleas", n)
}

func announceLine(n int) string {
	if n == 1 {
		return "1 pelea anunciada"
	}
	return fmt.Sprintf("%d peleas anunciadas", n)
}

func errorView(err error) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styleAccent.Render("ufc"),
		"",
		styleValue.Render(err.Error()),
		"",
		styleMuted.Render("q para salir"),
	)
}
