package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/model"

	"github.com/charmbracelet/lipgloss"
)

var (
	monthShort = map[time.Month]string{
		time.January:   "ene",
		time.February:  "feb",
		time.March:     "mar",
		time.April:     "abr",
		time.May:       "may",
		time.June:      "jun",
		time.July:      "jul",
		time.August:    "ago",
		time.September: "sep",
		time.October:   "oct",
		time.November:  "nov",
		time.December:  "dic",
	}

	monthLong = map[time.Month]string{
		time.January:   "enero",
		time.February:  "febrero",
		time.March:     "marzo",
		time.April:     "abril",
		time.May:       "mayo",
		time.June:      "junio",
		time.July:      "julio",
		time.August:    "agosto",
		time.September: "septiembre",
		time.October:   "octubre",
		time.November:  "noviembre",
		time.December:  "diciembre",
	}

	weekdayShort = map[time.Weekday]string{
		time.Sunday:    "dom",
		time.Monday:    "lun",
		time.Tuesday:   "mar",
		time.Wednesday: "mié",
		time.Thursday:  "jue",
		time.Friday:    "vie",
		time.Saturday:  "sáb",
	}
)

func dateLong(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d", weekdayShort[t.Weekday()], t.Day(), monthShort[t.Month()], t.Year())
}

func dateShort(t time.Time) string {
	return fmt.Sprintf("%s, %d %s", weekdayShort[t.Weekday()], t.Day(), monthShort[t.Month()])
}

func monthYear(t time.Time) string {
	return fmt.Sprintf("%s %d", monthLong[t.Month()], t.Year())
}

func dayShort(t time.Time) string {
	return fmt.Sprintf("%02d %s", t.Day(), weekdayShort[t.Weekday()])
}

func (m Model) nextView(width int) string {
	e, ok := dataset.Next(m.dataset)
	if !ok {
		return emptyState("no hay eventos próximos", "corre: go run ./cmd/ufc-sync")
	}

	if width < 96 {
		return nextStackedView(e, width, m.cursor)
	}

	leftWidth := (width - 3) * 50 / 100
	rightWidth := width - leftWidth - 3

	chrome := lipgloss.Height(m.headerView(width)) + lipgloss.Height(m.helpView()) + 2
	panelHeight := m.height - chrome
	if panelHeight < 16 {
		panelHeight = 16
	}

	left := upcomingPanel(e, leftWidth, panelHeight)
	right := fightCardPanel(e, rightWidth, panelHeight, m.cursor)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right)
}

func nextStackedView(e model.Event, width, cursor int) string {
	lines := []string{upcomingPanel(e, width, 0), ""}
	lines = append(lines, fightCardPanel(e, width, 0, cursor))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func upcomingPanel(e model.Event, width, height int) string {
	innerWidth := max(1, width-6)
	heading := panelHeading("PRÓXIMO EVENTO", "")
	eventName := styleTitle.Render(truncate(e.Name, innerWidth))

	dateCol := lipgloss.JoinVertical(
		lipgloss.Left,
		styleSection.Render("FECHA"),
		styleValue.Render(dateLong(e.Date)),
	)
	locCol := lipgloss.JoinVertical(
		lipgloss.Left,
		styleSection.Render("LUGAR"),
		styleValue.Render(truncate(e.Location, max(10, innerWidth-20))),
	)
	metaRow := lipgloss.JoinHorizontal(lipgloss.Top, dateCol, "      ", locCol)

	content := []string{
		heading,
		"",
		eventName,
		"",
		metaRow,
	}

	if main, ok := dataset.MainEvent(e); ok {
		content = append(content, "", mainEventBox(main.Fighters, main.WeightClass))
	}
	content = append(content, "", styleMuted.Render(announceLine(len(e.Card))))

	return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

func fightCardPanel(e model.Event, width, height, cursor int) string {
	innerWidth := max(1, width-6)
	heading := panelHeading("CARTELERA", fmt.Sprintf("(%d peleas)", len(e.Card)))
	if len(e.Card) == 0 {
		content := []string{heading, "", styleMuted.Render("cartelera no anunciada")}
		return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
	}

	content := []string{heading, ""}

	totalFights := len(e.Card)
	start := 0
	end := totalFights

	if height > 4 {
		contentH := height - 4
		slots := contentH - 2 // -2 para heading y línea en blanco
		if slots > 0 && totalFights > slots {
			visibleSlots := slots - 1
			if visibleSlots < 1 {
				visibleSlots = 1
			}
			start, end = window(totalFights, visibleSlots, cursor)
		}
	}

	weightW := 0
	for _, f := range e.Card {
		if w := lipgloss.Width(f.WeightClass); w > weightW {
			weightW = w
		}
	}
	if weightW > 18 {
		weightW = 18
	}

	fixed := 8 // num + espacios + vs
	availName := innerWidth - fixed - weightW
	nameW := availName / 2
	if nameW < 10 {
		nameW = 10
	}

	for i := start; i < end; i++ {
		fight := e.Card[i]
		var left, right string
		if len(fight.Fighters) > 0 {
			left = fight.Fighters[0]
		}
		if len(fight.Fighters) > 1 {
			right = fight.Fighters[1]
		}

		num := styleMuted.Render(fmt.Sprintf("%2d  ", i+1))
		names := styleValue.Render(truncate(left, nameW)) + "  " +
			styleMainEventVS.Render("vs") + "  " +
			styleValue.Render(truncate(right, nameW))

		row := num + names
		if fight.WeightClass != "" && innerWidth > 42 {
			remaining := max(1, innerWidth-lipgloss.Width(row)-lipgloss.Width(fight.WeightClass))
			row += strings.Repeat(" ", remaining) + styleMuted.Render(truncate(fight.WeightClass, weightW))
		}
		content = append(content, row)
	}

	if end-start < totalFights {
		info := fmt.Sprintf("mostrando %d-%d de %d  (↑/↓ desplazarse)", start+1, end, totalFights)
		content = append(content, "", styleMuted.Render(info))
	}

	return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

const tonightFootnote = "fechas según tu hora local"

func (m Model) tonightView(width int) string {
	lines := []string{
		panelHeading("HOY", "HORA LOCAL"),
		styleMuted.Render(repeat("─", max(1, width-6))),
		"",
	}

	if e, ok := dataset.Tonight(m.dataset); ok {
		lines = append(lines,
			styleAccent.Render("sí")+styleValue.Render("  "+e.Name),
			"",
			field("lugar", e.Location),
		)
		if main, has := dataset.MainEvent(e); has {
			lines = append(lines, "", mainEventBox(main.Fighters, main.WeightClass))
		}
	} else {
		lines = append(lines, styleMuted.Render("no")+styleValue.Render("  no hay pelea hoy"), "")
		if next, has := dataset.Next(m.dataset); has {
			d := daysUntil(next.Date)
			lines = append(lines,
				field("próximo", next.Name),
				field("cuándo", fmt.Sprintf("%s  (en %s)", dateShort(next.Date), pluralDays(d))),
			)
		}
	}

	lines = append(lines, "", styleMuted.Render(tonightFootnote))

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) calendarView(width int) string {
	up := dataset.Upcoming(m.dataset)
	if len(up) == 0 {
		return emptyState("no hay eventos próximos", "corre: go run ./cmd/ufc-sync")
	}

	lines := []string{
		panelHeading("CALENDARIO", "PRÓXIMOS EVENTOS"),
		styleMuted.Render(repeat("─", max(1, width-6))),
		"",
	}
	var lastMonth string

	for _, e := range up {
		month := monthYear(e.Date)
		if month != lastMonth {
			if lastMonth != "" {
				lines = append(lines, "")
			}
			lines = append(lines, styleAccent.Render(strings.ToUpper(month)))
			lastMonth = month
		}

		line := styleLabel.Render(dayShort(e.Date)) + "  " + styleValue.Render(e.Name)
		if n := len(e.Card); n > 0 {
			line += "  " + styleMuted.Render(fmt.Sprintf("(%d peleas)", n))
		}
		lines = append(lines, lipgloss.NewStyle().MaxWidth(width).Render(line))
	}

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func emptyState(title, hint string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styleTitle.Render(title),
		"",
		styleMuted.Render(hint),
	)
}

func pluralDays(n int) string {
	if n == 1 {
		return "1 día"
	}
	return fmt.Sprintf("%d días", n)
}

const (
	cardHeader = 3
	cardFooter = 2
)

const minCardHeight = 15

func (m Model) cardRows(width int) int {
	chrome := lipgloss.Height(m.headerView(width)) + lipgloss.Height(m.helpView()) + 2
	rows := m.height - chrome - cardHeader - cardFooter
	if rows < 1 {
		return 1
	}
	return rows
}

func window(total, slots, cursor int) (int, int) {
	if total <= 0 || slots <= 0 {
		return 0, 0
	}
	if slots >= total {
		return 0, total
	}

	start := clampCursor(cursor, total) - slots + 1
	if start > total-slots {
		start = total - slots
	}
	if start < 0 {
		start = 0
	}
	return start, start + slots
}

func cardColumns(e model.Event, width int) (nameW, weightW int) {
	for _, f := range e.Card {
		if w := lipgloss.Width(f.WeightClass); w > weightW {
			weightW = w
		}
	}

	if weightW > 20 {
		weightW = 20
	}

	const fixed = 17
	avail := width - fixed - weightW
	if avail < 16 {
		return 8, weightW
	}

	// El nombre más largo entre los peleadores para evitar espacios vacíos
	maxName := 0
	for _, f := range e.Card {
		for _, name := range f.Fighters {
			if w := lipgloss.Width(name); w > maxName {
				maxName = w
			}
		}
	}

	desired := maxName
	if desired < 12 {
		desired = 12
	}
	if desired > 22 {
		desired = 22
	}

	if avail/2 < desired {
		nameW = avail / 2
	} else {
		nameW = desired
	}

	if nameW < 8 {
		nameW = 8
	}
	return nameW, weightW
}

func cursorMarker(selected bool) string {
	if selected {
		return styleCursor.Render("▸ ")
	}
	return "  "
}

func fightRow(f model.Fight, idx int, selected bool, nameW, weightW, width int) string {
	marker := cursorMarker(selected)
	star := "   "
	if f.IsMainEvent {
		star = styleMainEventLabel.Render("☆  ")
	}

	var left, right string
	if len(f.Fighters) > 0 {
		left = f.Fighters[0]
	}
	if len(f.Fighters) > 1 {
		right = f.Fighters[1]
	}

	line := marker +
		" " + styleMuted.Render(fmt.Sprintf("%2d", idx+1)) +
		"  " + styleValue.Render(pad(truncate(left, nameW), nameW)) +
		"  " + styleMainEventVS.Render("vs") +
		"  " + styleValue.Render(pad(truncate(right, nameW), nameW)) +
		"  " + star +
		styleLabel.Render(pad(truncate(f.WeightClass, weightW), weightW))

	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}

	var b strings.Builder
	b.Grow(n * 2)
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > n-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}

func eventMeta(e model.Event) string {
	if e.Location == "" {
		return dateLong(e.Date)
	}
	return dateLong(e.Date) + "  -  " + e.Location
}

func (m Model) cardView(width int) string {
	e, ok := dataset.Next(m.dataset)
	if !ok {
		return emptyState("no hay eventos próximos", "corré: go run ./cmd/ufc-sync")
	}
	if len(e.Card) == 0 {
		return emptyState(e.Name, "cartelera no anunciada")
	}

	cursor := clampCursor(m.cursor, len(e.Card))
	nameW, weightW := cardColumns(e, width)

	lines := []string{
		panelHeading("CARTELERA", fmt.Sprintf("%d PELEAS - %s", len(e.Card), e.Name)),
		styleMuted.Render(eventMeta(e)),
		"",
	}

	start, end := window(len(e.Card), m.cardRows(width), cursor)
	for i := start; i < end; i++ {
		lines = append(lines, fightRow(e.Card[i], i, i == cursor, nameW, weightW, width))
	}

	lines = append(lines, "", cursorLine(cursor, len(e.Card), start, end))

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

const resultsLimit = 20

func (m Model) resultsView(width int) string {
	events := dataset.RecentResults(m.dataset, resultsLimit)
	if len(events) == 0 {
		return emptyState("sin resultados", "corré: go run ./cmd/ufc-sync")
	}

	cursor := clampCursor(m.cursor, len(events))

	lines := []string{
		panelHeading("RESULTADOS", "ÚLTIMOS PRIMERO"),
		styleMuted.Render(repeat("─", max(1, width-6))),
		"",
	}

	start, end := window(len(events), m.resultRows(width), cursor)
	for i := start; i < end; i++ {
		lines = append(lines, resultRow(events[i], i == cursor, width))
	}

	lines = append(lines, "", resultCursorLine(cursor, len(events), start, end))

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) resultRows(width int) int {
	chrome := lipgloss.Height(m.headerView(width)) + lipgloss.Height(m.helpView()) + 2
	rows := m.height - chrome - cardHeader - cardFooter
	if rows < 1 {
		return 1
	}
	return rows
}

func resultRow(e model.Event, selected bool, width int) string {
	marker := cursorMarker(selected)

	fixed := lipgloss.Width("  ") + 1 + lipgloss.Width(dateShort(time.Time{})) + 2
	detail := headlineResult(e)
	nameW := width - fixed - lipgloss.Width(detail) - 1
	if nameW < 12 {
		nameW = 12
	}
	if detail == "" {
		nameW = width - fixed
	}

	line := marker + " " +
		styleMuted.Render(dateShort(e.Date)) + "  " +
		styleValue.Render(truncate(e.Name, nameW)) + "  " +
		styleAccent.Render(detail)

	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

func headlineResult(e model.Event) string {
	f, ok := dataset.MainEvent(e)
	if !ok {
		return ""
	}
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

func resultCursorLine(cursor, total, start, end int) string {
	pos := fmt.Sprintf("evento %d de %d", cursor+1, total)
	if start > 0 || end < total {
		pos += fmt.Sprintf("   mostrando %d-%d", start+1, end)
	}
	if total > 1 {
		pos += "   ↑/↓ moverse"
	}
	return styleMuted.Render(pos)
}

func cursorLine(cursor, total, start, end int) string {
	pos := fmt.Sprintf("pelea %d de %d", cursor+1, total)
	if start > 0 || end < total {
		pos += fmt.Sprintf("   mostrando %d-%d", start+1, end)
	}
	if total > 1 {
		pos += "   ↑/↓ moverse"
	}
	return styleMuted.Render(pos)
}
