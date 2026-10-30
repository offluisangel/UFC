package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/export"
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

var viewNames = []string{"next", "tonight", "calendar", "card", "results"}

type Model struct {
	dataset model.Dataset
	current view
	cursor  int
	width   int
	height  int
	ready   bool
	err     error

	exportDir string
	notice    string
	noticeOK  bool
}

func New(ds model.Dataset, err error) Model {
	return Model{dataset: ds, err: err}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) goTo(v view) Model {
	if v != m.current {
		m.current = v
		m.cursor = 0
	}
	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		m.notice = ""
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "1":
			m = m.goTo(viewNext)
		case "2":
			m = m.goTo(viewTonight)
		case "3":
			m = m.goTo(viewCalendar)
		case "4":
			m = m.goTo(viewCard)
		case "5":
			m = m.goTo(viewResults)
		case "right", "l", "tab":
			m = m.goTo((m.current + 1) % numViews)
		case "left", "h", "shift+tab":
			m = m.goTo((m.current + numViews - 1) % numViews)
		case "up", "k":
			m.cursor--
		case "down", "j":
			m.cursor++
		case "e":
			m.notice, m.noticeOK = m.exportNext()
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = m.listLen() - 1
		}
		m.cursor = clampCursor(m.cursor, m.listLen())
	}
	return m, nil
}

func (m Model) exportNext() (msg string, ok bool) {
	e, has := dataset.Next(m.dataset)
	if !has {
		return "No upcoming event to export", false
	}
	path := filepath.Join(m.exportDir, export.Filename(e))
	if err := os.WriteFile(path, export.ICS(e, time.Now()), 0o644); err != nil {
		return "Failed to save .ics: " + err.Error(), false
	}
	return "Saved " + path, true
}

func (m Model) listLen() int {
	switch m.current {
	case viewNext, viewCard:
		e, ok := dataset.Next(m.dataset)
		if !ok {
			return 0
		}
		return len(e.Card)
	case viewCalendar:
		return len(dataset.Upcoming(m.dataset))
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
		return "Loading..."
	}
	if m.err != nil {
		return errorView(m.err)
	}

	width := m.width
	if width < 20 {
		width = 20
	}

	header := m.headerView(width)
	help := m.helpView()
	bodyHeight := m.bodyHeight(width)

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

func (m Model) bodyHeight(width int) int {
	h := m.height - lipgloss.Height(m.headerView(width)) - lipgloss.Height(m.helpView()) - 2
	if h < 1 {
		return 1
	}
	return h
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

func (m Model) tabsRow(width int) string {
	build := func(compact bool) string {
		tabs := make([]string, 0, numViews)
		for i, n := range viewNames {
			num := strconv.Itoa(i + 1)
			switch {
			case view(i) == m.current:
				tabs = append(tabs, styleTabActive.Render(" "+num+" "+n+" "))
			case compact:
				tabs = append(tabs, " "+styleTabNum.Render(num)+" ")
			default:
				tabs = append(tabs, " "+styleTabNum.Render(num)+" "+styleTabIdle.Render(n)+" ")
			}
		}
		return strings.Join(tabs, "")
	}

	row := build(false)
	if lipgloss.Width(row) > width {
		row = build(true)
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(row)
}

func (m Model) headerView(width int) string {
	brand := styleBadge.Render("UFC") + " " + styleTitle.Render("fight nights")
	count := styleMuted.Render(fmt.Sprintf("%d upcoming events", len(dataset.Upcoming(m.dataset))))

	titleRow := brand
	if gap := width - lipgloss.Width(brand) - lipgloss.Width(count); gap >= 2 {
		titleRow = brand + strings.Repeat(" ", gap) + count
	}
	titleRow = lipgloss.NewStyle().MaxWidth(width).Render(titleRow)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleRow,
		m.tabsRow(width),
		rule(width),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) helpView() string {
	sep := styleDim.Render("  |  ")
	hint := func(key, txt string) string {
		return styleHelpKey.Render(key) + styleHelpTxt.Render(" "+txt)
	}

	if m.notice != "" {
		mark := styleAccent.Render("! ")
		if m.noticeOK {
			mark = styleAccent.Render("✓ ")
		}
		return lipgloss.NewStyle().MaxWidth(m.helpWidth()).Render(mark + styleValue.Render(m.notice))
	}

	jump := hint("1-5", "go to view")
	switchTab := hint("←/→", "switch views")
	quit := hint("q", "quit")

	var move string
	switch m.current {
	case viewCard, viewNext:
		move = hint("↑/↓", "fights")
	case viewCalendar, viewResults:
		move = hint("↑/↓", "events")
	}

	var exportKey string
	switch m.current {
	case viewNext, viewTonight, viewCard:
		exportKey = hint("e", "export .ics")
	}

	candidates := [][]string{
		{jump, switchTab, move, exportKey, quit},
		{switchTab, move, exportKey, quit},
		{switchTab, move, quit},
		{switchTab, quit},
		{quit},
	}
	for _, c := range candidates {
		parts := make([]string, 0, len(c))
		for _, p := range c {
			if p != "" {
				parts = append(parts, p)
			}
		}
		line := strings.Join(parts, sep)
		if lipgloss.Width(line) <= m.helpWidth() {
			return line
		}
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
	return strings.Repeat(s, n)
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

func pluralDays(n int) string {
	if n == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", n)
}

func countdown(days int) string {
	switch {
	case days <= 0:
		return "TODAY"
	case days == 1:
		return "TOMORROW"
	default:
		return fmt.Sprintf("IN %d DAYS", days)
	}
}

func announceLine(n int) string {
	if n == 1 {
		return "1 fight announced"
	}
	return fmt.Sprintf("%d fights announced", n)
}

func errorView(err error) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styleAccent.Render("ufc"),
		"",
		styleValue.Render(err.Error()),
		"",
		styleMuted.Render("q to quit"),
	)
}
