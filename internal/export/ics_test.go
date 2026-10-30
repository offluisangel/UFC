package export

import (
	"strings"
	"testing"
	"time"

	"github.com/offluisangel/UFC/internal/model"
)

var stamp = time.Date(2026, time.October, 21, 12, 30, 0, 0, time.UTC)

func sampleEvent() model.Event {
	return model.Event{
		ID:       "abc123",
		Name:     "UFC 333: Volkanovski vs. Evloev",
		Date:     time.Date(2026, time.October, 24, 0, 0, 0, 0, time.UTC),
		Location: "Abu Dhabi, Abu Dhabi, United Arab Emirates",
		Status:   model.StatusUpcoming,
		Card: []model.Fight{
			{Fighters: []string{"Alexander Volkanovski", "Movsar Evloev"}, WeightClass: "Featherweight", IsMainEvent: true},
			{Fighters: []string{"Petr Yan", "Merab Dvalishvili"}, WeightClass: "Bantamweight"},
		},
	}
}

func TestICSIsAnAllDayEvent(t *testing.T) {
	out := string(ICS(sampleEvent(), stamp))

	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n",
		"VERSION:2.0\r\n",
		"BEGIN:VEVENT\r\n",
		"UID:abc123@ufc-tui\r\n",
		"DTSTAMP:20261021T123000Z\r\n",
		"DTSTART;VALUE=DATE:20261024\r\n",
		"DTEND;VALUE=DATE:20261025\r\n",
		"TRANSP:TRANSPARENT\r\n",
		"END:VEVENT\r\n",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "DTSTART:") || strings.Contains(out, "T00") {
		t.Errorf("event must not carry a time of day:\n%s", out)
	}
}

func TestICSDescribesTheEvent(t *testing.T) {
	out := string(ICS(sampleEvent(), stamp))

	for _, want := range []string{
		`SUMMARY:UFC 333: Volkanovski vs. Evloev`,
		`LOCATION:Abu Dhabi\, Abu Dhabi\, United Arab Emirates`,
		`Main event: Alexander Volkanovski vs Movsar Evloev (Featherweight)`,
		`2 fights announced.`,
		`Event time not included`,
	} {
		if !strings.Contains(unfold(out), want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestICSOmitsEmptyLocation(t *testing.T) {
	e := sampleEvent()
	e.Location = ""
	if out := string(ICS(e, stamp)); strings.Contains(out, "LOCATION") {
		t.Errorf("empty location was exported:\n%s", out)
	}
}

func TestICSUnannouncedCard(t *testing.T) {
	e := sampleEvent()
	e.Card = nil
	out := unfold(string(ICS(e, stamp)))
	if !strings.Contains(out, "Fight card not announced.") {
		t.Errorf("unannounced card not mentioned:\n%s", out)
	}
	if strings.Contains(out, "Main event") {
		t.Errorf("invented a main event:\n%s", out)
	}
}

func TestICSIgnoresClockAndZoneOfTheDate(t *testing.T) {
	e := sampleEvent()
	e.Date = time.Date(2026, time.October, 24, 23, 30, 0, 0, time.FixedZone("x", -5*3600))
	out := string(ICS(e, stamp))
	if !strings.Contains(out, "DTSTART;VALUE=DATE:20261024\r\n") || !strings.Contains(out, "DTEND;VALUE=DATE:20261025\r\n") {
		t.Errorf("date shifted by the clock or zone:\n%s", out)
	}
}

func TestICSEndRollsOverMonths(t *testing.T) {
	e := sampleEvent()
	e.Date = time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	if out := string(ICS(e, stamp)); !strings.Contains(out, "DTEND;VALUE=DATE:20270101\r\n") {
		t.Errorf("end did not roll over the year:\n%s", out)
	}
}

func TestUIDIsStableAcrossExports(t *testing.T) {
	e := sampleEvent()
	a := string(ICS(e, stamp))
	b := string(ICS(e, stamp.Add(48*time.Hour)))
	if line(a, "UID:") != line(b, "UID:") {
		t.Errorf("UID changed between exports: %q vs %q", line(a, "UID:"), line(b, "UID:"))
	}

	e.ID = ""
	got := line(string(ICS(e, stamp)), "UID:")
	if got != "UID:ufc-333-volkanovski-vs-evloev-20261024@ufc-tui" {
		t.Errorf("fallback UID = %q", got)
	}
}

func TestEscape(t *testing.T) {
	cases := map[string]string{
		"plain":       "plain",
		"a, b":        `a\, b`,
		"a; b":        `a\; b`,
		`back\slash`:  `back\\slash`,
		"two\nlines":  `two\nlines`,
		"win\r\nline": `win\nline`,
	}
	for in, want := range cases {
		if got := escape(in); got != want {
			t.Errorf("escape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldKeepsLinesShortAndRunesWhole(t *testing.T) {
	long := "DESCRIPTION:" + strings.Repeat("áé€ ", 60)
	folded := fold(long)

	for i, l := range strings.Split(folded, "\r\n") {
		if len(l) > foldLimit {
			t.Errorf("line %d is %d octets, limit %d", i, len(l), foldLimit)
		}
		if i > 0 && !strings.HasPrefix(l, " ") {
			t.Errorf("continuation line %d does not start with a space: %q", i, l)
		}
		if !isValidUTF8(l) {
			t.Errorf("line %d splits a rune: %q", i, l)
		}
	}
	if got := unfold(folded); got != long {
		t.Errorf("unfolding does not restore the original line")
	}
	if short := "SUMMARY:x"; fold(short) != short {
		t.Errorf("short line was folded")
	}
}

func TestICSLinesAllEndWithCRLFAndFit(t *testing.T) {
	e := sampleEvent()
	e.Name = strings.Repeat("Un nombre larguísimo, con comas; y símbolos ", 5)
	out := string(ICS(e, stamp))

	if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") {
		t.Errorf("bare LF found:\n%q", out)
	}
	for i, l := range strings.Split(strings.TrimSuffix(out, "\r\n"), "\r\n") {
		if len(l) > foldLimit {
			t.Errorf("line %d is %d octets: %q", i, len(l), l)
		}
	}
}

func TestFilename(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"UFC 333: Volkanovski vs. Evloev", "ufc-333-volkanovski-vs-evloev-2026-10-24.ics"},
		{"  UFC Fight Night: Moicano vs. Nolan!! ", "ufc-fight-night-moicano-vs-nolan-2026-10-24.ics"},
		{"", "ufc-event-2026-10-24.ics"},
		{"???", "ufc-event-2026-10-24.ics"},
	}
	for _, c := range cases {
		e := sampleEvent()
		e.Name = c.name
		if got := Filename(e); got != c.want {
			t.Errorf("Filename(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func unfold(s string) string {
	return strings.ReplaceAll(s, "\r\n ", "")
}

func line(out, prefix string) string {
	for _, l := range strings.Split(unfold(out), "\r\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}
