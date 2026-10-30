package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/model"
)

func testDataset(t *testing.T) model.Dataset {
	t.Helper()
	path := filepath.Join("..", "..", "data", "events.json")
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no dataset at %s: %v", path, err)
	}
	ds, err := dataset.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(ds.Events) == 0 && len(buf) == 0 {
		t.Fatal("empty dataset")
	}
	ds.SyncedAt = time.Now()
	return ds
}

func press(t *testing.T, m Model, k string) Model {
	t.Helper()
	var msg tea.KeyMsg
	switch k {
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	nm, _ := m.Update(msg)
	next, ok := nm.(Model)
	if !ok {
		t.Fatalf("key %q returned %T, want ui.Model", k, nm)
	}
	return next
}

func renderAt(t *testing.T, m Model, w, h int) string {
	t.Helper()
	var out string
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	out = next.View()
	if strings.Contains(out, "cargando...") {
		t.Fatal("view not ready after WindowSizeMsg")
	}
	return out
}

func TestViewsRender(t *testing.T) {
	ds := testDataset(t)

	for v := range viewNames {
		m := Model{dataset: ds, current: view(v)}
		out := renderAt(t, m, 80, 30)
		if strings.TrimSpace(out) == "" {
			t.Fatalf("view %v rendered empty", viewNames[v])
		}
		t.Logf("\n===== %s =====\n%s", viewNames[v], out)
	}
}

func resultsDataset(t *testing.T, n int) model.Dataset {
	t.Helper()
	events := make([]model.Event, 0, n+1)
	events = append(events, model.Event{
		ID:       "next",
		Name:     "UFC Futura",
		Date:     time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC),
		Location: "Las Vegas, Nevada, USA",
		Status:   model.StatusUpcoming,
		Card: []model.Fight{{
			Fighters:    []string{"Futuro", "Invicto"},
			WeightClass: "Heavyweight",
			IsMainEvent: true,
		}},
	})
	for i := 0; i < n; i++ {
		day := 1 + i
		events = append(events, model.Event{
			ID:       fmt.Sprintf("done-%d", i),
			Name:     fmt.Sprintf("UFC Pasado %d", i+1),
			Date:     time.Date(2026, time.September, day, 0, 0, 0, 0, time.UTC),
			Location: "Las Vegas, Nevada, USA",
			Status:   model.StatusCompleted,
			Card: []model.Fight{{
				Fighters:    []string{"Ganador", "Perdedor"},
				WeightClass: "Middleweight",
				Method:      "KO",
				Round:       "2",
				Time:        "3:12",
				Winner:      "Ganador",
				IsMainEvent: true,
			}},
		})
	}
	return model.Dataset{Events: events}
}

func TestViewSwitching(t *testing.T) {
	ds := testDataset(t)
	m := Model{dataset: ds, width: 80, height: 30}

	keys := []string{"2", "3", "4", "5", "1", "right", "right", "right", "right", "right", "left"}
	want := []view{
		viewTonight, viewCalendar, viewCard, viewResults, viewNext,
		viewTonight, viewCalendar, viewCard, viewResults, viewNext, viewResults,
	}

	for i, k := range keys {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		m = nm.(Model)
		if m.current != want[i] {
			t.Fatalf("key %q: got view %v, want %v", k, viewNames[m.current], viewNames[want[i]])
		}
	}
}

func TestCursorBoundFollowsTheView(t *testing.T) {
	ds := resultsDataset(t, 6)
	next := ds.Events[0]
	for i := 0; i < 12; i++ {
		next.Card = append(next.Card, model.Fight{
			Fighters: []string{"A", "B"},
			Method:   "KO",
		})
	}
	ds.Events = append([]model.Event{next}, ds.Events[1:]...)

	card := Model{dataset: ds, current: viewCard, width: 80, height: 30}
	results := Model{dataset: ds, current: viewResults, width: 80, height: 30}

	const fights = 13
	if got := card.listLen(); got != fights {
		t.Errorf("cartelera bound = %d, want %d", got, fights)
	}
	if got, want := results.listLen(), 6; got != want {
		t.Errorf("results bound = %d, want %d", got, want)
	}

	m := results
	for i := 0; i < 20; i++ {
		m = press(t, m, "down")
	}
	if m.cursor != 5 {
		t.Errorf("results cursor = %d, want it clamped to the last event (5)", m.cursor)
	}

	m = card
	for i := 0; i < 20; i++ {
		m = press(t, m, "down")
	}
	if m.cursor != fights-1 {
		t.Errorf("cartelera cursor = %d, want it clamped to the last fight (%d)", m.cursor, fights-1)
	}
}

func TestQuitKeys(t *testing.T) {
	ds := testDataset(t)
	for _, k := range []string{"q", "ctrl+c", "esc"} {
		m := Model{dataset: ds, width: 80, height: 30}
		nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		if cmd == nil {
			t.Fatalf("key %q did not produce a quit command", k)
		}
		if _, ok := nm.(Model); !ok {
			t.Fatalf("key %q returned wrong type", k)
		}
	}
}

func TestNoViewShowsATimeOfDay(t *testing.T) {
	ds := testDataset(t)
	clock := regexp.MustCompile(`\b\d{1,2}:\d{2}\b`)

	for _, v := range []view{viewNext, viewTonight, viewCalendar, viewCard} {
		out := renderAt(t, Model{dataset: ds, current: v}, 80, 30)

		if m := clock.FindString(out); m != "" {
			t.Errorf("view %v renders a time of day (%q):\n%s", viewNames[v], m, out)
		}
		if strings.Contains(out, "start time") {
			t.Errorf("view %v still renders an %q field:\n%s", viewNames[v], "inicio", out)
		}
		if strings.Contains(out, "0001-01-01") {
			t.Errorf("view %v leaked a zero time:\n%s", viewNames[v], out)
		}
	}
}

func TestErrorViewRenders(t *testing.T) {
	m := Model{err: os.ErrNotExist, width: 80, height: 30}
	out := renderAt(t, m, 80, 30)
	if !strings.Contains(out, "ufc") || !strings.Contains(out, "q to quit") {
		t.Fatalf("error view not informative:\n%s", out)
	}
	t.Logf("\n===== error =====\n%s", out)
}

func TestClampCursor(t *testing.T) {
	cases := []struct {
		cursor, n, want int
	}{
		{0, 0, 0},   // empty card: no valid index
		{5, 0, 0},   // empty card, stale cursor
		{-1, 10, 0}, // above the start
		{0, 10, 0},
		{3, 10, 3},
		{9, 10, 9},
		{10, 10, 9}, // one past the end
		{99, 10, 9},
	}
	for _, c := range cases {
		if got := clampCursor(c.cursor, c.n); got != c.want {
			t.Errorf("clampCursor(%d, %d) = %d, want %d", c.cursor, c.n, got, c.want)
		}
	}
}

func TestWindowKeepsCursorVisible(t *testing.T) {
	cases := []struct {
		total, slots, cursor int
		wantStart, wantEnd   int
	}{
		{13, 13, 0, 0, 13},  // fits whole
		{13, 13, 12, 0, 13}, // fits whole, cursor at the end
		{13, 5, 0, 0, 5},    // top of the list
		{13, 5, 2, 0, 5},    // still fits from the top
		{13, 5, 5, 1, 6},    // cursor pushed the window down one
		{13, 5, 12, 8, 13},  // pinned to the end
		{13, 5, 99, 8, 13},  // cursor clamped, then scrolled
		{13, 5, -3, 0, 5},   // cursor clamped, then top aligned
		{1, 5, 0, 0, 1},     // one fight
		{0, 5, 0, 0, 0},     // no fights
		{13, 0, 0, 0, 0},    // no room
	}

	for _, c := range cases {
		start, end := window(c.total, c.slots, c.cursor)
		if start != c.wantStart || end != c.wantEnd {
			t.Errorf("window(%d, %d, %d) = (%d, %d), want (%d, %d)",
				c.total, c.slots, c.cursor, start, end, c.wantStart, c.wantEnd)
		}
		if c.slots <= 0 || c.total <= 0 {
			continue
		}
		if end-start != min(c.slots, c.total) {
			t.Errorf("window(%d, %d, %d) drew %d rows, want %d",
				c.total, c.slots, c.cursor, end-start, min(c.slots, c.total))
		}
		if got := clampCursor(c.cursor, c.total); got < start || got >= end {
			t.Errorf("window(%d, %d, %d) = (%d, %d) hides cursor at %d",
				c.total, c.slots, c.cursor, start, end, got)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"Islam Makhachev", 20, "Islam Makhachev"}, // already fits
		{"Islam Makhachev", 16, "Islam Makhachev"},
		{"Islam Makhachev", 5, "Isla…"},
		{"Jack Della Maddalena", 6, "Jack …"},
		{"abc", 0, ""},
		{"abc", -1, ""},
		{"abc", 1, "…"},
		{"", 5, ""},
	}
	for _, c := range cases {
		got := truncate(c.in, c.n)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
		if w := lipgloss.Width(got); w > c.n && c.n > 0 {
			t.Errorf("truncate(%q, %d) is %d columns wide", c.in, c.n, w)
		}
	}
}

func TestPadMeasuresColumnsNotBytes(t *testing.T) {
	for _, label := range []string{"date", "location", "start", "when", "next"} {
		if got := lipgloss.Width(pad(label, 12)); got != 12 {
			t.Errorf("pad(%q, 12) is %d columns, want 12", label, got)
		}
	}
	if got := pad("un-label-largo", 4); got != "un-label-largo" {
		t.Errorf("pad shrank a %q", got)
	}
}

func TestFightRowNeverExceedsWidth(t *testing.T) {
	long := model.Fight{
		Fighters:    []string{"Alexandra An Extraordinary Long Name", "Bartholomew The Second Longest"},
		WeightClass: "Light Heavyweight",
		IsMainEvent: true,
	}
	short := model.Fight{Fighters: []string{"A vs. B", "C"}, WeightClass: "Flyweight"}
	oneSided := model.Fight{Fighters: []string{"Solo"}, WeightClass: "Welterweight"}

	for _, width := range []int{20, 40, 60, 80, 100, 120, 200} {
		for name, f := range map[string]model.Fight{"long": long, "short": short, "one-sided": oneSided} {
			e := model.Event{Card: []model.Fight{f}}
			nameW, weightW := cardColumns(e, width)
			row := fightRow(f, 0, true, nameW, weightW, width)
			if got := lipgloss.Width(row); got > width {
				t.Errorf("width %d, %s card: row is %d columns", width, name, got)
			}
			if strings.Contains(row, "\n") {
				t.Errorf("width %d, %s card: row wrapped", width, name)
			}
		}
	}
}

func TestCursorLine(t *testing.T) {
	if got := cursorLine(0, 13, 0, 13); !strings.Contains(got, "fight 1 of 13") {
		t.Errorf("missing position: %q", got)
	}

	got := cursorLine(12, 13, 8, 13)
	if !strings.Contains(got, "fight 13 of 13") || !strings.Contains(got, "showing 9-13") {
		t.Errorf("scrolled footer incomplete: %q", got)
	}
	if got := cursorLine(0, 1, 0, 1); strings.Contains(got, "move") {
		t.Errorf("single-fight card offers to move: %q", got)
	}
}

func TestCardViewRenders(t *testing.T) {
	ds := testDataset(t)
	e, ok := dataset.Next(ds)
	if !ok {
		t.Skip("no upcoming events")
	}

	m := Model{dataset: ds, current: viewCard}
	out := renderAt(t, m, 80, 30)

	if !strings.Contains(out, e.Name) {
		t.Errorf("card view missing the event name %q:\n%s", e.Name, out)
	}
	if !strings.Contains(out, "vs") {
		t.Errorf("card view has no fights:\n%s", out)
	}
	if !strings.Contains(out, "▸") {
		t.Errorf("no cursor marker on the first fight:\n%s", out)
	}
	if !strings.Contains(out, "☆") {
		t.Errorf("main event not flagged:\n%s", out)
	}
	if !strings.Contains(out, "fight 1 of") {
		t.Errorf("missing position counter:\n%s", out)
	}
	t.Logf("\n===== cartelera =====\n%s", out)
}

func TestCardCursorNavigation(t *testing.T) {
	ds := testDataset(t)
	m := Model{dataset: ds, current: viewCard, width: 80, height: 30}
	n := m.listLen()
	if n < 2 {
		t.Skip("card too short to navigate")
	}

	for i := 1; i < n; i++ {
		m = press(t, m, "down")
		if m.cursor != i {
			t.Fatalf("after %d downs: cursor %d, want %d", i, m.cursor, i)
		}
	}
	if m = press(t, m, "down"); m.cursor != n-1 {
		t.Errorf("cursor ran past the end: %d, want %d", m.cursor, n-1)
	}

	for i := n - 2; i >= 0; i-- {
		m = press(t, m, "up")
		if m.cursor != i {
			t.Fatalf("after %d ups: cursor %d, want %d", n-1-i, m.cursor, i)
		}
	}
	if m = press(t, m, "up"); m.cursor != 0 {
		t.Errorf("cursor ran before the start: %d", m.cursor)
	}

	if m = press(t, m, "j"); m.cursor != 1 {
		t.Errorf("j did not move down: %d", m.cursor)
	}
	if m = press(t, m, "k"); m.cursor != 0 {
		t.Errorf("k did not move up: %d", m.cursor)
	}
}

func TestCardViewFitsWindow(t *testing.T) {
	ds := testDataset(t)
	e, ok := dataset.Next(ds)
	if !ok || len(e.Card) < 5 {
		t.Skip("need a card long enough to scroll")
	}
	for _, h := range []int{40, 30, 24, 20, 18, 16, minCardHeight} {
		m := Model{dataset: ds, current: viewCard, cursor: len(e.Card) - 1}
		out := renderAt(t, m, 80, h)

		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) > h {
			t.Errorf("height %d: rendered %d lines", h, len(lines))
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w > 80 {
				t.Errorf("height %d line %d: %d columns", h, i+1, w)
			}
		}
		if !strings.Contains(out, fmt.Sprintf("fight %d of %d", len(e.Card), len(e.Card))) {
			t.Errorf("height %d: last fight not scrolled into view:\n%s", h, out)
		}
	}

	for _, h := range []int{minCardHeight - 1, 12, 10, 5, 1} {
		m := Model{dataset: ds, current: viewCard}
		out := renderAt(t, m, 80, h)
		if !strings.Contains(out, "fight 1 of") {
			t.Errorf("height %d: cart view lost its content:\n%s", h, out)
		}
	}
}

func TestCardViewEmptyCard(t *testing.T) {
	ds := model.Dataset{Events: []model.Event{{
		ID:   "x",
		Name: "UFC Test: Empty",
		Date: dataset.Today().Add(48 * time.Hour),
	}}}
	m := Model{dataset: ds, current: viewCard}
	out := renderAt(t, m, 80, 30)

	if !strings.Contains(out, "Fight card not announced") {
		t.Fatalf("expected empty-card message, got:\n%s", out)
	}
	if m := press(t, m, "down"); m.cursor != 0 {
		t.Errorf("cursor moved on an empty card: %d", m.cursor)
	}
}

func TestNoViewOverflowsTheWidth(t *testing.T) {
	ds := resultsDataset(t, 6)
	widths := []int{120, 100, 80, 60, 48, 40, 32, 24, 20}

	for _, w := range widths {
		for v := range viewNames {
			m := Model{dataset: ds, current: view(v), width: w, height: 40}
			out := renderAt(t, m, w, 40)
			for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
				if got := lipgloss.Width(line); got > w {
					t.Errorf("width %d, view %s, line %d: %d columns", w, viewNames[v], i+1, got)
				}
			}
		}
	}
}

func TestHeaderShowsEveryTabNumber(t *testing.T) {
	for _, w := range []int{80, 60, 40, 24} {
		for v := range viewNames {
			m := Model{width: w, height: 30, current: view(v)}
			got := stripANSI(m.tabsRow(w))
			for i := range viewNames {
				if w >= 40 && !strings.Contains(got, strconv.Itoa(i+1)) {
					t.Errorf("width %d, view %s: tabs dropped number %d: %q", w, viewNames[v], i+1, got)
				}
			}
			if w >= 40 && !strings.Contains(got, viewNames[v]) {
				t.Errorf("width %d: active tab %q lost its name: %q", w, viewNames[v], got)
			}
		}
	}
}

func TestHelpLineDegradesWithoutLosingQuit(t *testing.T) {
	for _, w := range []int{80, 60, 48, 40, 32, 24, 20} {
		for v := range viewNames {
			m := Model{width: w, height: 30, current: view(v)}
			got := stripANSI(m.helpView())
			if !strings.Contains(got, "q quit") {
				t.Errorf("width %d, view %s: lost the quit hint: %q", w, viewNames[v], got)
			}
			if got := lipgloss.Width(m.helpView()); got > w && w >= 20 {
				t.Errorf("width %d, view %s: help is %d columns", w, viewNames[v], got)
			}
		}
	}
}

func indexOfView(name string) int {
	for i, n := range viewNames {
		if n == name {
			return i
		}
	}
	return 0
}

func TestHelpLineShowsCardKeysOnCardView(t *testing.T) {
	ds := testDataset(t)
	m := Model{dataset: ds, width: 80, height: 30}

	if got := m.helpView(); !strings.Contains(got, "←/→ switch views") {
		t.Errorf("non-card view lost the tab hint: %q", got)
	}
	if got := press(t, m, "4").helpView(); !strings.Contains(got, "↑/↓ fights") {
		t.Errorf("card view does not advertise the fight keys: %q", got)
	}
	if got := press(t, m, "5").helpView(); !strings.Contains(got, "↑/↓ events") {
		t.Errorf("results view does not advertise the event keys: %q", got)
	}
}

func TestResultsViewRenders(t *testing.T) {
	m := Model{dataset: resultsDataset(t, 6), current: viewResults, width: 80, height: 30}
	out := renderAt(t, m, 80, 30)

	if !strings.Contains(out, "▸") {
		t.Errorf("no cursor marker on the first event:\n%s", out)
	}
	if !strings.Contains(out, "event 1 of 6") {
		t.Errorf("missing position counter:\n%s", out)
	}
	// The headline result is the reason this view exists.
	if !strings.Contains(out, "KO R2 3:12") {
		t.Errorf("result row is missing the method, round and time:\n%s", out)
	}
	t.Logf("\n===== resultados =====\n%s", out)
}

func TestResultsViewIsNewestFirst(t *testing.T) {
	ds := resultsDataset(t, 4)
	m := Model{dataset: ds, current: viewResults, width: 80, height: 30}
	out := renderAt(t, m, 80, 30)
	hi := strings.Index(out, "UFC Pasado 4")
	lo := strings.Index(out, "UFC Pasado 1")
	if hi < 0 || lo < 0 {
		t.Fatalf("not every event rendered:\n%s", out)
	}
	if hi > lo {
		t.Errorf("results are not newest first:\n%s", out)
	}
}

func TestResultRowsNeverWrap(t *testing.T) {
	ds := resultsDataset(t, 3)
	long := model.Event{
		ID:     "long",
		Name:   "UFC Con Un Nombre De Evento Muchisimo Mas Largo De Lo Normal Y Con Mas Palabras Todavia",
		Date:   time.Date(2026, time.September, 9, 0, 0, 0, 0, time.UTC),
		Status: model.StatusCompleted,
		Card: []model.Fight{{
			Fighters:    []string{"Un Peleador Con Nombre Larguisimo", "Otro Peleador Con Nombre Larguisimo"},
			WeightClass: "Light Heavyweight",
			Method:      "Decision (Unanimous)",
			Round:       "5",
			Time:        "5:00",
			Winner:      "Un Peleador Con Nombre Larguisimo",
			IsMainEvent: true,
		}},
	}
	ds.Events = append([]model.Event{long}, ds.Events...)

	for _, w := range []int{120, 100, 80, 60, 40, 24} {
		for _, selected := range []bool{true, false} {
			row := resultRow(long, selected, w)
			if strings.Contains(row, "\n") {
				t.Errorf("width %d selected %v: row wrapped:\n%q", w, selected, row)
			}
			if got := lipgloss.Width(row); got > w {
				t.Errorf("width %d selected %v: row is %d columns", w, selected, got)
			}
		}
	}
}

func TestSelectedAndUnselectedRowsShareTheirColumns(t *testing.T) {
	e := model.Event{
		Name:   "UFC 300",
		Date:   time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
		Card:   []model.Fight{{Method: "KO", Round: "2", Time: "3:12", IsMainEvent: true}},
		Status: model.StatusCompleted,
	}

	sel := stripANSI(resultRow(e, true, 80))
	unsel := stripANSI(resultRow(e, false, 80))

	const date = "Thu, 3 Sep"
	si, ui := strings.Index(sel, date), strings.Index(unsel, date)
	if si < 0 || ui < 0 {
		t.Fatalf("date missing from a row:\n%q\n%q", sel, unsel)
	}
	sc, uc := lipgloss.Width(sel[:si]), lipgloss.Width(unsel[:ui])
	if sc != uc {
		t.Errorf("date starts at column %d when selected and %d when not:\n%q\n%q", sc, uc, sel, unsel)
	}
	if lipgloss.Width(sel) != lipgloss.Width(unsel) {
		t.Errorf("rows are %d and %d columns wide", lipgloss.Width(sel), lipgloss.Width(unsel))
	}
	if !strings.Contains(sel, "▸") {
		t.Error("the selected row has no cursor marker")
	}
}

func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

func TestResultsViewFitsWindow(t *testing.T) {
	ds := resultsDataset(t, 20)
	for _, h := range []int{40, 30, 24, 20, 18, 16, minCardHeight} {
		m := Model{dataset: ds, current: viewResults, cursor: 19, width: 80, height: h}
		out := renderAt(t, m, 80, h)

		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) > h {
			t.Errorf("height %d: rendered %d lines", h, len(lines))
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w > 80 {
				t.Errorf("height %d line %d: %d columns", h, i+1, w)
			}
		}
		if !strings.Contains(out, "event 20 of 20") {
			t.Errorf("height %d: last event not scrolled into view:\n%s", h, out)
		}
	}
}

func TestResultsViewEmpty(t *testing.T) {
	ds := model.Dataset{Events: []model.Event{{
		ID:     "next",
		Name:   "UFC Futura",
		Date:   dataset.Today().Add(72 * time.Hour),
		Status: model.StatusUpcoming,
	}}}
	m := Model{dataset: ds, current: viewResults, width: 80, height: 30}
	out := renderAt(t, m, 80, 30)

	if !strings.Contains(out, "No results") {
		t.Errorf("expected an empty-state message, got:\n%s", out)
	}
}

func TestHeadlineResult(t *testing.T) {
	tests := []struct {
		name string
		card []model.Fight
		want string
	}{
		{
			"method, round and time",
			[]model.Fight{{Method: "KO", Round: "2", Time: "3:12", IsMainEvent: true}},
			"KO R2 3:12",
		},
		{
			"method without a time",
			[]model.Fight{{Method: "Submission", Round: "1", IsMainEvent: true}},
			"Submission R1",
		},
		{
			"no result yet",
			[]model.Fight{{IsMainEvent: true}},
			"",
		},
		{
			"no main event flagged",
			[]model.Fight{{Method: "KO", Round: "1"}},
			"",
		},
		{
			"empty card",
			nil,
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := model.Event{Name: "x", Card: tc.card}
			if got := headlineResult(e); got != tc.want {
				t.Errorf("headlineResult = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHeadlineResultPrefersTheMainEvent(t *testing.T) {
	e := model.Event{Card: []model.Fight{
		{Method: "Decision", Round: "1", IsMainEvent: false},
		{Method: "TKO", Round: "3", Time: "4:05", Winner: "B", IsMainEvent: true},
		{Method: "Guillotine", Round: "2", IsMainEvent: false},
	}}
	if got, want := headlineResult(e), "TKO R3 4:05"; got != want {
		t.Errorf("headlineResult = %q, want %q", got, want)
	}
}

func TestUnannouncedCardStaysVisibleInEveryView(t *testing.T) {
	ds := model.Dataset{Events: []model.Event{{
		ID:       "fa2fca260beeb9c0",
		Name:     "UFC 335",
		Date:     time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC),
		Location: "Las Vegas, Nevada, USA",
		Status:   model.StatusUpcoming,
	}}}

	for v, name := range viewNames {
		for _, w := range []int{100, 80, 60} {
			m := Model{dataset: ds, current: view(v)}
			out := renderAt(t, m, w, 30)
			if name != "results" && !strings.Contains(out, "UFC 335") {
				t.Errorf("view %q at width %d hides an event that is on the calendar:\n%s", name, w, out)
			}
			for i, line := range strings.Split(out, "\n") {
				if n := len([]rune(line)); n > w {
					t.Errorf("view %q at width %d: line %d is %d cols: %q", name, w, i, n, line)
				}
			}
		}
	}

	m := Model{dataset: ds, current: viewCard}
	if out := renderAt(t, m, 80, 30); !strings.Contains(out, "Fight card not announced") {
		t.Errorf("card view should explain the empty card, got:\n%s", out)
	}
}
