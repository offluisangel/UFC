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
		time.January:   "Jan",
		time.February:  "Feb",
		time.March:     "Mar",
		time.April:     "Apr",
		time.May:       "May",
		time.June:      "Jun",
		time.July:      "Jul",
		time.August:    "Aug",
		time.September: "Sep",
		time.October:   "Oct",
		time.November:  "Nov",
		time.December:  "Dec",
	}

	monthLong = map[time.Month]string{
		time.January:   "January",
		time.February:  "February",
		time.March:     "March",
		time.April:     "April",
		time.May:       "May",
		time.June:      "June",
		time.July:      "July",
		time.August:    "August",
		time.September: "September",
		time.October:   "October",
		time.November:  "November",
		time.December:  "December",
	}

	weekdayShort = map[time.Weekday]string{
		time.Sunday:    "Sun",
		time.Monday:    "Mon",
		time.Tuesday:   "Tue",
		time.Wednesday: "Wed",
		time.Thursday:  "Thu",
		time.Friday:    "Fri",
		time.Saturday:  "Sat",
	}
)

const (
	cardFooter      = 2
	minCardHeight   = 15
	wideLayout      = 96
	resultsLimit    = 20
	tonightFootnote = "dates shown in your local time"
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

func (m Model) listRows(width, headerLines int) int {
	rows := m.bodyHeight(width) - headerLines - cardFooter
	if rows < 1 {
		return 1
	}
	return rows
}

func emptyState(title, hint string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styleTitle.Render(title),
		"",
		styleMuted.Render(hint),
	)
}

// ---------------------------------------------------------------- next

func (m Model) nextView(width int) string {
	e, ok := dataset.Next(m.dataset)
	if !ok {
		return emptyState("No upcoming events", "Run: go run ./cmd/ufc-sync")
	}
	cursor := clampCursor(m.cursor, len(e.Card))

	if width < wideLayout {
		up := upcomingPanel(e, width, 0, true)
		fightsH := 0
		if m.height >= minCardHeight {
			fightsH = max(8, m.bodyHeight(width)-lipgloss.Height(up)-1)
		}
		return lipgloss.JoinVertical(lipgloss.Left, up, "", fightCardPanel(e, width, fightsH, cursor))
	}

	const gap = 2
	leftW := (width - gap) / 2
	rightW := width - leftW - gap
	panelH := max(16, m.bodyHeight(width))

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		upcomingPanel(e, leftW, panelH, false),
		strings.Repeat(" ", gap),
		fightCardPanel(e, rightW, panelH, cursor),
	)
}

func upcomingPanel(e model.Event, width, height int, compact bool) string {
	iw := max(1, width-6)
	days := daysUntil(e.Date)

	var content []string
	if compact {
		content = []string{
			headerLine("NEXT EVENT", countdown(days), iw),
			styleTitle.Render(truncate(e.Name, iw)),
			styleValue.Render(truncate(dateLong(e.Date), iw)),
			styleMuted.Render(truncate(e.Location, iw)),
		}
	} else {
		content = []string{
			headerLine("NEXT EVENT", "", iw),
			"",
			styleTitle.Render(truncate(e.Name, iw)),
			styleAccent.Render(countdown(days)),
			"",
			field("date", truncate(dateLong(e.Date), max(1, iw-12))),
			field("location", truncate(e.Location, max(1, iw-12))),
		}
		if main, ok := dataset.MainEvent(e); ok {
			box := mainEventBox(main.Fighters, main.WeightClass)
			content = append(content, "", lipgloss.NewStyle().MaxWidth(iw).Render(box))
		}
		content = append(content, "", styleMuted.Render(announceLine(len(e.Card))))
	}

	return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

func fightCardPanel(e model.Event, width, height, cursor int) string {
	iw := max(1, width-6)
	heading := headerLine("FIGHT CARD", fmt.Sprintf("%d fights", len(e.Card)), iw)

	if len(e.Card) == 0 {
		content := []string{heading, "", styleMuted.Render("Fight card not announced")}
		return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
	}

	total := len(e.Card)
	start, end := 0, total
	if height > 4 {
		slots := height - 4 - 2 // heading + blank line
		if slots > 0 && total > slots {
			visible := max(1, slots-2) // blank line + scroll hint
			start, end = window(total, visible, cursor)
		}
	}

	nameW, weightW := cardColumns(e, iw)
	if iw < 64 {
		weightW = 0
		nameW = max(6, (iw-14)/2)
	}

	content := []string{heading, ""}
	for i := start; i < end; i++ {
		content = append(content, fightRow(e.Card[i], i, i == cursor, nameW, weightW, iw))
	}
	if end-start < total {
		hint := fmt.Sprintf("Showing %d-%d of %d", start+1, end, total)
		content = append(content, "", styleMuted.Render(hint))
	}

	return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

// ---------------------------------------------------------------- tonight

func (m Model) tonightView(width int) string {
	lines := []string{
		sectionHeader("TONIGHT", "LOCAL TIME", width),
		"",
	}

	if e, ok := dataset.Tonight(m.dataset); ok {
		lines = append(lines,
			styleBadge.Render("FIGHTS TONIGHT"),
			"",
			styleTitle.Render(truncate(e.Name, width)),
			styleMuted.Render(truncate(e.Location, width)),
		)
		if main, has := dataset.MainEvent(e); has {
			box := mainEventBox(main.Fighters, main.WeightClass)
			lines = append(lines, "", lipgloss.NewStyle().MaxWidth(width).Render(box))
		}
	} else {
		lines = append(lines, styleTitle.Render("No fights tonight"), "")
		if next, has := dataset.Next(m.dataset); has {
			lines = append(lines,
				styleAccent.Render(countdown(daysUntil(next.Date))),
				"",
				field("next", truncate(next.Name, max(1, width-12))),
				field("when", truncate(fmt.Sprintf("%s  (in %s)", dateShort(next.Date), pluralDays(daysUntil(next.Date))), max(1, width-12))),
			)
		}
	}

	lines = append(lines, "", styleDim.Render(tonightFootnote))

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// ------------------------------------------------------------- calendar

const calendarHeader = 3

func (m Model) calendarView(width int) string {
	up := dataset.Upcoming(m.dataset)
	if len(up) == 0 {
		return emptyState("No upcoming events", "Run: go run ./cmd/ufc-sync")
	}
	cursor := clampCursor(m.cursor, len(up))

	var (
		lines      []string
		cursorLine int
		lastMonth  string
	)
	for i, e := range up {
		if month := monthYear(e.Date); month != lastMonth {
			if lastMonth != "" {
				lines = append(lines, "")
			}
			lines = append(lines, styleAccent.Render(strings.ToUpper(month)))
			lastMonth = month
		}
		if i == cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, calendarRow(e, i == cursor, width))
	}

	start, end := 0, len(lines)
	if m.height >= minCardHeight {
		start, end = window(len(lines), m.listRows(width, calendarHeader), cursorLine)
	}

	out := []string{sectionHeader("CALENDAR", "UPCOMING EVENTS", width), ""}
	out = append(out, lines[start:end]...)
	out = append(out, "", positionLine("evento", cursor, len(up), 0, len(up), len(up)))

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, out...))
}

func calendarRow(e model.Event, selected bool, width int) string {
	const (
		prefixW = 2 + 6 + 2
		countW  = 13
		locW    = 24
	)

	count := "No card"
	if n := len(e.Card); n > 0 {
		count = announce(n)
	}

	showLoc := width >= 84
	nameW := width - prefixW - countW - 2
	if showLoc {
		nameW -= locW + 2
	}
	nameW = max(8, nameW)

	nameStyle := styleValue
	if selected {
		nameStyle = styleMainEventName
	}

	line := cursorMarker(selected) +
		cell(styleLabel, dayShort(e.Date), selected) + spaces(2, selected) +
		cell(nameStyle, pad(truncate(e.Name, nameW), nameW), selected) + spaces(2, selected)
	if showLoc {
		line += cell(styleMuted, pad(truncate(e.Location, locW), locW), selected) + spaces(2, selected)
	}
	line += cell(styleDim, repeat(" ", max(0, countW-lipgloss.Width(count)))+count, selected)

	return fillRow(line, width, selected)
}

func announce(n int) string {
	if n == 1 {
		return "1 fight"
	}
	return fmt.Sprintf("%d fights", n)
}

// -------------------------------------------------------------- fight card

const cardHeader = 4

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

	const fixed = 14 // marker + star + number + gaps + " vs "
	avail := width - fixed - weightW
	if avail < 16 {
		return 8, weightW
	}

	maxName := 0
	for _, f := range e.Card {
		for _, name := range f.Fighters {
			if w := lipgloss.Width(name); w > maxName {
				maxName = w
			}
		}
	}

	desired := min(max(maxName, 12), 22)
	nameW = desired
	if avail/2 < desired {
		nameW = avail / 2
	}
	return max(nameW, 8), weightW
}

func cursorMarker(selected bool) string {
	if selected {
		return cell(styleCursor, "▸ ", true)
	}
	return "  "
}

func fightRow(f model.Fight, idx int, selected bool, nameW, weightW, width int) string {
	left, right := fighterNames(f.Fighters)

	nameStyle := styleValue
	if selected {
		nameStyle = styleMainEventName
	}

	star := spaces(2, selected)
	if f.IsMainEvent {
		star = cell(styleStar, "☆ ", selected)
	}

	line := cursorMarker(selected) + star +
		cell(styleDim, fmt.Sprintf("%2d", idx+1), selected) + spaces(2, selected) +
		cell(nameStyle, pad(truncate(left, nameW), nameW), selected) +
		cell(styleMainEventVS, " vs ", selected) +
		cell(nameStyle, pad(truncate(right, nameW), nameW), selected)
	if weightW > 0 {
		line += spaces(2, selected) + cell(styleLabel, pad(truncate(f.WeightClass, weightW), weightW), selected)
	}

	return fillRow(line, width, selected)
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
		return emptyState("No upcoming events", "Run: go run ./cmd/ufc-sync")
	}
	if len(e.Card) == 0 {
		return emptyState(e.Name, "Fight card not announced")
	}

	cursor := clampCursor(m.cursor, len(e.Card))
	nameW, weightW := cardColumns(e, width)

	lines := []string{
		sectionHeader("FIGHT CARD", fmt.Sprintf("%d fights - %s", len(e.Card), e.Name), width),
		styleMuted.Render(truncate(eventMeta(e), width)),
		"",
	}

	start, end := window(len(e.Card), m.listRows(width, cardHeader), cursor)
	for i := start; i < end; i++ {
		lines = append(lines, fightRow(e.Card[i], i, i == cursor, nameW, weightW, width))
	}

	lines = append(lines, "", cursorLine(cursor, len(e.Card), start, end))

	return lipgloss.NewStyle().Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// ------------------------------------------------------------- results

const (
	resultsHeader = 3
	resultDateW   = 11
)

func (m Model) resultsView(width int) string {
	events := dataset.RecentResults(m.dataset, resultsLimit)
	if len(events) == 0 {
		return emptyState("No results", "Run: go run ./cmd/ufc-sync")
	}

	cursor := clampCursor(m.cursor, len(events))

	const gap = 3
	wide := width >= 120
	listW := width
	if wide {
		listW = (width - gap) * 58 / 100
	}

	lines := []string{
		sectionHeader("RESULTS", "NEWEST FIRST", listW),
		"",
	}

	start, end := window(len(events), m.listRows(width, resultsHeader), cursor)
	for i := start; i < end; i++ {
		lines = append(lines, resultRow(events[i], i == cursor, listW))
	}
	lines = append(lines, "", resultCursorLine(cursor, len(events), start, end))

	list := lipgloss.NewStyle().Width(listW).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	if !wide {
		return list
	}

	detail := resultDetailPanel(events[cursor], width-listW-gap, max(12, m.bodyHeight(width)))
	return lipgloss.JoinHorizontal(lipgloss.Top, list, strings.Repeat(" ", gap), detail)
}

func resultRow(e model.Event, selected bool, width int) string {
	prefixW := 2 + resultDateW + 2
	detail := headlineResult(e)

	nameW := width - prefixW
	if detail != "" {
		nameW -= lipgloss.Width(detail) + 2
	}
	nameW = max(12, nameW)

	nameStyle := styleValue
	if selected {
		nameStyle = styleMainEventName
	}

	line := cursorMarker(selected) +
		cell(styleMuted, pad(dateShort(e.Date), resultDateW), selected) + spaces(2, selected) +
		cell(nameStyle, pad(truncate(e.Name, nameW), nameW), selected)
	if detail != "" {
		line += spaces(2, selected) + cell(styleAccent, detail, selected)
	}

	return fillRow(line, width, selected)
}

func resultDetailPanel(e model.Event, width, height int) string {
	iw := max(1, width-6)

	content := []string{
		headerLine("DETAILS", dateShort(e.Date), iw),
		"",
		styleTitle.Render(truncate(e.Name, iw)),
		styleMuted.Render(truncate(e.Location, iw)),
		"",
	}

	room := max(1, height-4-len(content))
	shown := len(e.Card)
	if shown > room {
		shown = max(1, room-1)
	}
	for _, f := range e.Card[:shown] {
		content = append(content, fightResultLine(f, iw))
	}
	if shown < len(e.Card) {
		content = append(content, styleDim.Render(fmt.Sprintf("+ %d more fights", len(e.Card)-shown)))
	}
	if len(e.Card) == 0 {
		content = append(content, styleMuted.Render("No card data available"))
	}

	return panelStyleH(width, height, false).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

func fightResultLine(f model.Fight, width int) string {
	left, right := fighterNames(f.Fighters)
	outcome := fightOutcome(f)

	star := "  "
	if f.IsMainEvent {
		star = styleStar.Render("☆ ")
	}

	leftSt, rightSt := styleValue, styleValue
	sep := " vs "
	switch {
	case f.Winner != "" && f.Winner == left:
		leftSt, rightSt, sep = styleMainEventName, styleMuted, " def. "
	case f.Winner != "" && f.Winner == right:
		left, right = right, left
		leftSt, rightSt, sep = styleMainEventName, styleMuted, " def. "
	}

	budget := width - 2 - lipgloss.Width(sep)
	if outcome != "" {
		budget -= lipgloss.Width(outcome) + 2
	}
	left, right = fitPair(left, right, budget)

	names := leftSt.Render(left) + styleDim.Render(sep) + rightSt.Render(right)
	line := star + names
	if outcome != "" {
		gap := max(2, width-lipgloss.Width(line)-lipgloss.Width(outcome))
		line += strings.Repeat(" ", gap) + styleAccent.Render(outcome)
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

func fitPair(a, b string, budget int) (string, string) {
	budget = max(2, budget)
	wa, wb := lipgloss.Width(a), lipgloss.Width(b)
	if wa+wb <= budget {
		return a, b
	}
	half := budget / 2
	switch {
	case wa <= half:
		return a, truncate(b, budget-wa)
	case wb <= budget-half:
		return truncate(a, budget-wb), b
	default:
		return truncate(a, half), truncate(b, budget-half)
	}
}

func headlineResult(e model.Event) string {
	f, ok := dataset.MainEvent(e)
	if !ok {
		return ""
	}
	return fightOutcome(f)
}

func positionLine(noun string, cursor, total, start, end, shown int) string {
	pos := fmt.Sprintf("%s %d of %d", noun, cursor+1, total)
	if start > 0 || end < shown {
		pos += fmt.Sprintf("   showing %d-%d", start+1, end)
	}
	if total > 1 {
		pos += "   ↑/↓ move"
	}
	return styleMuted.Render(pos)
}

func resultCursorLine(cursor, total, start, end int) string {
	return positionLine("event", cursor, total, start, end, total)
}

func cursorLine(cursor, total, start, end int) string {
	return positionLine("fight", cursor, total, start, end, total)
}
