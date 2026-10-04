package fetch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offluisangel/UFC/internal/model"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	buf, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return string(buf)
}

func TestParseEventLinksFixture(t *testing.T) {
	links, err := parseEventLinks(fixture(t, "upcoming.html"))
	if err != nil {
		t.Fatalf("parseEventLinks: %v", err)
	}
	if len(links) == 0 {
		t.Fatal("no events parsed from fixture")
	}

	seen := map[string]bool{}
	for _, l := range links {
		if l.ID == "" {
			t.Errorf("event with empty id: %+v", l)
		}
		if l.Name == "" {
			t.Errorf("event %s has empty name", l.ID)
		}
		if seen[l.ID] {
			t.Errorf("duplicate id %s", l.ID)
		}
		seen[l.ID] = true
	}
	t.Logf("parsed %d upcoming events", len(links))
	for _, l := range links {
		t.Logf("  %s  %s", l.ID, l.Name)
	}
}

func TestParseEventLinksEmptyIsError(t *testing.T) {
	if _, err := parseEventLinks("<html><body>nothing here</body></html>"); err == nil {
		t.Fatal("expected error on HTML with no event links, got nil")
	}
}

func TestParseEventLinksIgnoresChallenge(t *testing.T) {
	body := `<html><body>Checking your browser.<script>var nonce="abc";</script></body></html>`
	if _, err := parseEventLinks(body); err == nil {
		t.Fatal("challenge page must not parse as a valid event list")
	}
}

func TestParseEventDetailUpcoming(t *testing.T) {
	html := fixture(t, "event_upcoming.html")

	ev, err := parseEventDetail("fixture-id", html)
	if err != nil {
		t.Fatalf("parseEventDetail: %v", err)
	}

	if ev.Name == "" {
		t.Error("name is empty")
	}
	if ev.Date.IsZero() {
		t.Error("date is zero")
	}
	if ev.Date.Hour() != 0 || ev.Date.Minute() != 0 || ev.Date.Second() != 0 {
		t.Errorf("date carries a time of day: %s", ev.Date)
	}
	if ev.Location == "" {
		t.Error("location is empty")
	}
	if len(ev.Card) == 0 {
		t.Fatal("card is empty: this is the exact failure mode of a broken selector")
	}

	main, ok := datasetMainEvent(ev)
	if !ok {
		t.Fatal("no main event found")
	}
	if len(main.Fighters) < 2 {
		t.Errorf("main event has %d fighters, want >= 2", len(main.Fighters))
	}
	if main.WeightClass == "" {
		t.Error("main event has no weight class")
	}
	if !main.IsMainEvent {
		t.Error("main fight not flagged")
	}

	t.Logf("%s | %s | %s", ev.Name, ev.Date.Format("2006-01-02"), ev.Location)
	t.Logf("main: %s (%s)", strings.Join(main.Fighters, " vs "), main.WeightClass)
	t.Logf("card size: %d", len(ev.Card))
}

func TestParseEventDetailCompleted(t *testing.T) {
	html := fixture(t, "event_completed.html")

	ev, err := parseEventDetail("fixture-completed", html)
	if err != nil {
		t.Fatalf("parseEventDetail: %v", err)
	}
	if len(ev.Card) == 0 {
		t.Fatal("completed event produced an empty card")
	}

	withResult := 0
	for _, f := range ev.Card {
		if f.Method != "" {
			withResult++
			if f.Round == "" {
				t.Errorf("%s: method %q but no round", f.Fighters, f.Method)
			}
			if f.WeightClass == "" {
				t.Errorf("%s: method %q but no weight class", f.Fighters, f.Method)
			}
		}
	}
	if withResult == 0 {
		t.Fatal("no fight carried a method: result extraction is broken")
	}

	t.Logf("%s | %d fights | %d with results", ev.Name, len(ev.Card), withResult)
	f := ev.Card[0]
	t.Logf("  [%s] %s — %s R%s @ %s", f.WeightClass, strings.Join(f.Fighters, " vs "), f.Method, f.Round, f.Time)
}

func TestParseEventDetailErrors(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{"empty", "<html><body></body></html>"},
		{"no title", `<html><body><table class="b-fight-details__table"><tbody><tr><td><a href="/fighter-details/aaa">A</a><a href="/fighter-details/bbb">B</a></td></tr></tbody></table></body></html>`},
		{"title but no date", `<html><body><h2 class="b-content__title">X</h2></body></html>`},
		{"unparseable date", `<html><body><h2 class="b-content__title">X</h2><ul><li class="b-list__box-list-item"><i class="b-list__box-item-title">Date:</i> someday</li></ul></body></html>`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseEventDetail("x", tc.html); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestParseEventDetailDetectsMissingFightTable(t *testing.T) {
	html := `<html><body>
		<h2 class="b-content__title">UFC 999</h2>
		<ul><li class="b-list__box-list-item"><i class="b-list__box-item-title">Date:</i> October 03, 2026</li></ul>
		<table class="totally-different-table"><tbody><tr><td>nope</td></tr></tbody></table>
	</body></html>`

	_, err := parseEventDetail("x", html)
	if err == nil {
		t.Fatal("expected an error when the fight table is absent")
	}
	if !strings.Contains(err.Error(), "fight table") {
		t.Fatalf("error should name the missing fight table, got: %v", err)
	}
}

func TestParseEventDetailAnnouncedButEmptyCardIsNotAFault(t *testing.T) {
	html := `<html><body>
		<h2 class="b-content__title">UFC 335</h2>
		<ul>
			<li class="b-list__box-list-item"><i class="b-list__box-item-title">Date:</i> December 12, 2026</li>
			<li class="b-list__box-list-item"><i class="b-list__box-item-title">Location:</i> Las Vegas, Nevada, USA</li>
		</ul>
		<table class="b-fight-details__table b-fight-details__table_type_event-details js-fight-table">
			<thead><tr><th>W/L</th><th>Fighter</th><th>Kd</th><th>Str</th><th>Td</th><th>Sub</th><th>Method</th><th>Round</th><th>Time</th></tr></thead>
			<tbody></tbody>
		</table>
	</body></html>`

	ev, err := parseEventDetail("fa2fca260beeb9c0", html)
	if err != nil {
		t.Fatalf("an unannounced card is a normal state, not a parse failure: %v", err)
	}
	if len(ev.Card) != 0 {
		t.Fatalf("card should be empty, got %d fights", len(ev.Card))
	}
	if got := ev.Date.Format("2006-01-02"); got != "2026-12-12" {
		t.Fatalf("the event is real and its date must survive, got %q", got)
	}
	if ev.Location != "Las Vegas, Nevada, USA" {
		t.Fatalf("location must survive too, got %q", ev.Location)
	}
}

func TestParseEventDetailRowsWeCannotReadIsAFault(t *testing.T) {
	html := `<html><body>
		<h2 class="b-content__title">UFC 999</h2>
		<ul><li class="b-list__box-list-item"><i class="b-list__box-item-title">Date:</i> October 03, 2026</li></ul>
		<table class="b-fight-details__table"><tbody><tr><td>one cell only</td></tr></tbody></table>
	</body></html>`

	_, err := parseEventDetail("x", html)
	if err == nil {
		t.Fatal("expected an error when rows exist but none yields a fight")
	}
	if !strings.Contains(err.Error(), "selectors changed") {
		t.Fatalf("the error should blame the selectors, not an unannounced card, got: %v", err)
	}
}

func TestParseEventDate(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"October 03, 2026", "2026-10-03", true},
		{"January 02, 2006", "2006-01-02", true},
		{"Dec 25, 2024", "2024-12-25", true},
		{"March 8, 2025", "2025-03-08", true},
		{"", "", false},
		{"someday", "", false},
		{"October 03", "", false},
	}

	for _, tc := range tests {
		got, err := model.ParseEventDate(tc.in)
		if tc.ok != (err == nil) {
			t.Errorf("%q: ok mismatch, err=%v", tc.in, err)
			continue
		}
		if tc.ok && got.Format("2006-01-02") != tc.want {
			t.Errorf("%q: got %s, want %s", tc.in, got.Format("2006-01-02"), tc.want)
		}
	}
}

func foughtCard(row string) string {
	return `<html><body>
<h2 class="b-content__title">UFC 300</h2>
<ul><li class="b-list__box-list-item"><i class="b-list__box-item-title">Date:</i> May 03, 2024</li></ul>
<table class="b-fight-details__table">
<thead><tr><th>W/L</th><th>Fighter</th><th>Weight class</th><th>Method</th><th>Round</th><th>Time</th></tr></thead>
<tbody>` + row + `</tbody></table>
</body></html>`
}

func fightRow(flag, first, second string) string {
	wl := ""
	if flag != "" {
		wl = `<i class="b-flag__text">` + flag + `</i>`
	}
	return `<tr class="b-fight-details__table-row">
<td><p>` + wl + `</p></td>
<td><p><a href="/fighter-details/aaa">` + first + `</a></p><p><a href="/fighter-details/bbb">` + second + `</a></p></td>
<td>Middleweight</td>
<td>KO</td>
<td>2</td>
<td>4:30</td>
</tr>`
}

func TestParseEventDetailWinner(t *testing.T) {
	tests := []struct {
		name   string
		flag   string
		first  string
		second string
		want   string
	}{
		{"green win flag", "win", "Alpha", "Bravo", "Alpha"},
		{"red loss flag", "loss", "Alpha", "Bravo", "Bravo"},
		{"no flag means undecided", "", "Alpha", "Bravo", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := parseEventDetail("x", foughtCard(fightRow(tc.flag, tc.first, tc.second)))
			if err != nil {
				t.Fatalf("parseEventDetail: %v", err)
			}
			if len(ev.Card) != 1 {
				t.Fatalf("got %d fights, want 1", len(ev.Card))
			}
			if got := ev.Card[0].Winner; got != tc.want {
				t.Errorf("winner = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseEventDetailCompletedFixtureNamesAWinnerWhenThereWasOne(t *testing.T) {
	ev, err := parseEventDetail("fixture-completed", fixture(t, "event_completed.html"))
	if err != nil {
		t.Fatalf("parseEventDetail: %v", err)
	}

	withdrawn := 0
	for _, f := range ev.Card {
		if f.Method == "" {
			t.Errorf("%s: finished fight with no method", f.Fighters)
		}
		if f.Winner == "" {
			if !noDecision(f.Method) {
				t.Errorf("%s: method %q but no winner", f.Fighters, f.Method)
			}
			withdrawn++
			continue
		}
		if f.Winner != f.Fighters[0] {
			t.Errorf("%s: winner %q is not the first fighter, which is how this site orders it",
				f.Fighters, f.Winner)
		}
	}
	if withdrawn > 0 {
		t.Logf("%d of %d fights stopped with no decision", withdrawn, len(ev.Card))
	}
}

func noDecision(method string) bool {
	return strings.HasPrefix(method, "CNC") || strings.HasPrefix(method, "DQ")
}

func TestParseEventDetailUpcomingFixtureHasNoWinner(t *testing.T) {
	ev, err := parseEventDetail("fixture-upcoming", fixture(t, "event_upcoming.html"))
	if err != nil {
		t.Fatalf("parseEventDetail: %v", err)
	}
	for _, f := range ev.Card {
		if f.Winner != "" {
			t.Errorf("%s: upcoming fight claims winner %q", f.Fighters, f.Winner)
		}
	}
}

func datasetMainEvent(e model.Event) (model.Fight, bool) {
	for _, f := range e.Card {
		if f.IsMainEvent {
			return f, true
		}
	}
	return model.Fight{}, false
}
