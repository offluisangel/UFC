package dataset

import (
	"testing"
	"time"

	"github.com/offluisangel/UFC/internal/model"
)

func cal(y int, m time.Month, d int) time.Time {
	return DateOnly(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

func day(n int) time.Time { return cal(2026, time.October, n) }

func ev(name string, d time.Time) model.Event {
	return model.Event{Name: name, Date: d, Location: "Las Vegas, Nevada, USA"}
}

func fought(name string, d time.Time) model.Event {
	e := ev(name, d)
	e.Status = model.StatusCompleted
	return e
}

func ds(events ...model.Event) model.Dataset {
	return model.Dataset{Events: events}
}

func zoneApartFromUTC(t *testing.T) *time.Location {
	t.Helper()
	u := time.Now().UTC()
	for _, hours := range []int{-12, 14, -8, 9, -5, 5, -1, 1} {
		loc := time.FixedZone("test", hours*3600)
		l := u.In(loc)
		if l.Year() != u.Year() || l.YearDay() != u.YearDay() {
			return loc
		}
	}
	t.Fatal("no fixed offset separates the calendar dates at this instant")
	return nil
}

func TestTodayUsesTheLocalCalendarDate(t *testing.T) {
	orig := time.Local
	t.Cleanup(func() { time.Local = orig })
	time.Local = zoneApartFromUTC(t)

	now := time.Now()
	got := Today()

	if y, m, d := got.Date(); y != now.Year() || m != now.Month() || d != now.Day() {
		t.Errorf("Today() = %04d-%02d-%02d, want the local date %04d-%02d-%02d",
			y, m, d, now.Year(), now.Month(), now.Day())
	}
	if got.Location() != time.UTC {
		t.Errorf("Today() is in %v, want UTC midnight like every other calendar date", got.Location())
	}
	if h, m, s := got.Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("Today() is not normalised to midnight: %s", got)
	}
}

func TestLocalDateDiffersFromUTCDate(t *testing.T) {
	loc := zoneApartFromUTC(t)
	u := time.Now().UTC()
	l := u.In(loc)

	if l.Year() == u.Year() && l.YearDay() == u.YearDay() {
		t.Fatalf("test setup is broken: %v and %v share a calendar date", u, l)
	}
}

func TestTonightAsOfMatchesOnlyTheExactDate(t *testing.T) {
	d := ds(
		ev("ayer", day(2)),
		ev("hoy", day(3)),
		ev("mañana", day(4)),
	)

	got, ok := tonightAsOf(d, day(3))
	if !ok {
		t.Fatal("tonight: no event for the reference date")
	}
	if got.Name != "hoy" {
		t.Errorf("tonight: got %q, want %q", got.Name, "hoy")
	}

	// Yesterday's card is over and tomorrow's has not started.
	if _, ok := tonightAsOf(d, day(5)); ok {
		t.Error("tonight: matched an event on a different date")
	}
	if _, ok := tonightAsOf(d, day(1)); ok {
		t.Error("tonight: matched an event on a different date")
	}
	if _, ok := tonightAsOf(ds(), day(3)); ok {
		t.Error("tonight: matched on an empty dataset")
	}
}

func TestTonightAsOfIgnoresTheTimeOfDay(t *testing.T) {
	d := ds(ev("hoy", day(3)))
	ref := time.Date(2026, time.October, 3, 23, 59, 59, 0, time.FixedZone("lejos", 13*3600))

	if _, ok := tonightAsOf(d, ref); !ok {
		t.Error("tonight: a 23:59 reference on the same date did not match")
	}
}

func TestUpcomingAsOfDropsThePastAndSorts(t *testing.T) {
	d := ds(
		ev("d", day(9)),
		ev("pasado", day(1)),
		ev("b", day(5)),
		ev("a", day(3)),
	)

	up := upcomingAsOf(d, day(3))
	got := make([]string, 0, len(up))
	for _, e := range up {
		got = append(got, e.Name)
	}

	want := []string{"a", "b", "d"}
	if len(got) != len(want) {
		t.Fatalf("upcoming: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("upcoming: got %v, want %v", got, want)
		}
	}
}

func TestNextAsOfIgnoresInputOrder(t *testing.T) {
	d := ds(
		ev("lejano", day(20)),
		ev("pasado", day(1)),
		ev("proximo", day(4)),
		ev("medio", day(11)),
	)

	got, ok := nextAsOf(d, day(3))
	if !ok {
		t.Fatal("next: no event")
	}
	if got.Name != "proximo" {
		t.Errorf("next: got %q, want %q", got.Name, "proximo")
	}

	if _, ok := nextAsOf(ds(ev("pasado", day(1))), day(3)); ok {
		t.Error("next: returned an event that already happened")
	}
	if _, ok := nextAsOf(ds(), day(3)); ok {
		t.Error("next: returned an event from an empty dataset")
	}
}

func TestDateOnlyKeepsTheDayAndDropsTheRest(t *testing.T) {
	zones := []*time.Location{
		time.UTC,
		time.FixedZone("negativo", -5*3600),
		time.FixedZone("positivo", 14*3600),
	}
	for _, loc := range zones {
		in := time.Date(2026, time.October, 3, 22, 30, 15, 0, loc)
		got := DateOnly(in)

		if y, m, d := got.Date(); y != 2026 || m != time.October || d != 3 {
			t.Errorf("DateOnly(%v in %v) = %s, want 2026-10-03", in, loc, got)
		}
		if h, m, s := got.Clock(); h != 0 || m != 0 || s != 0 {
			t.Errorf("DateOnly kept a time of day: %s", got)
		}
		if !got.Equal(cal(2026, time.October, 3)) {
			t.Errorf("DateOnly(%v) is not the UTC midnight of its own day", in)
		}
	}
}

func TestUpcomingAsOfDropsCardsAlreadyFought(t *testing.T) {
	today := Today()
	past := DateOnly(today.AddDate(0, 0, -1))
	pending := DateOnly(today)

	d := ds(
		fought("pelado ayer", past),
		ev("esta noche", pending),
		fought("semana pasada", DateOnly(today.AddDate(0, 0, -7))),
	)

	up := upcomingAsOf(d, today)
	if len(up) != 1 {
		t.Fatalf("upcoming: got %d events, want only the pending one", len(up))
	}
	if up[0].Name != "esta noche" {
		t.Errorf("upcoming: got %q, want %q", up[0].Name, "esta noche")
	}
	if _, ok := nextAsOf(d, today); !ok {
		t.Error("next: a fought card left no next event")
	} else if n, _ := nextAsOf(d, today); n.Name != "esta noche" {
		t.Errorf("next: got %q, want the pending card", n.Name)
	}
}

func TestTonightAsOfReturnsACardDatedThatDayEvenIfResultsExist(t *testing.T) {
	today := DateOnly(cal(2026, time.October, 3))
	d := ds(
		fought("con resultados", today),
		ev("sin resultados", today),
	)

	got, ok := tonightAsOf(d, today)
	if !ok {
		t.Fatal("tonight: no event")
	}
	if got.Name != "con resultados" {
		t.Errorf("tonight: got %q, want the first card dated that day", got.Name)
	}
}

func TestRecentResultsIsNewestFirstAndCapped(t *testing.T) {
	today := Today()
	past := func(offset int) time.Time { return DateOnly(today.AddDate(0, 0, -offset)) }

	d := ds(
		fought("viejo", past(30)),
		ev("futuro", past(-30)),
		fought("reciente", past(2)),
		fought("medio", past(9)),
	)

	tests := []struct {
		limit int
		want  []string
	}{
		{0, []string{"reciente", "medio", "viejo"}},  // 0 means no cap
		{20, []string{"reciente", "medio", "viejo"}}, // asking for more than exists
		{2, []string{"reciente", "medio"}},
		{1, []string{"reciente"}},
	}

	for _, tc := range tests {
		got := RecentResults(d, tc.limit)
		names := make([]string, 0, len(got))
		for _, e := range got {
			names = append(names, e.Name)
		}
		if len(names) != len(tc.want) {
			t.Errorf("limit %d: got %v, want %v", tc.limit, names, tc.want)
			continue
		}
		for i := range tc.want {
			if names[i] != tc.want[i] {
				t.Errorf("limit %d: got %v, want %v", tc.limit, names, tc.want)
				break
			}
		}
	}
}

func TestRecentResultsExcludesPendingCards(t *testing.T) {
	d := ds(ev("viejo sin confirmar", day(1)))

	if got := RecentResults(d, 20); len(got) != 0 {
		t.Errorf("got %d results, want none", len(got))
	}
}

func TestRecentResultsOnAnEmptyDataset(t *testing.T) {
	if got := RecentResults(ds(), 20); len(got) != 0 {
		t.Errorf("got %d results from an empty dataset, want none", len(got))
	}
}

func TestFutureEventMarkedCompletedIsStillUpcoming(t *testing.T) {
	today := Today()
	future := DateOnly(today.AddDate(0, 0, 1))
	mislabelled := fought("UFC futuro", future)
	up := upcomingAsOf(ds(mislabelled), today)

	if len(up) != 1 {
		t.Fatalf("a future event marked completed vanished from upcoming: %d events", len(up))
	}
	if got := RecentResults(ds(mislabelled), 20); len(got) != 0 {
		t.Errorf("a future event was listed as a result: %d", len(got))
	}
}

func TestEventDatedTodayStaysUpcoming(t *testing.T) {
	today := Today()
	e := fought("UFC de hoy", DateOnly(today))

	if isFought(e, today) {
		t.Error("a card dated today must not be reported as a result")
	}
	if up := upcomingAsOf(ds(e), today); len(up) != 1 {
		t.Errorf("a card dated today dropped out of upcoming: %d", len(up))
	}
	if got := RecentResults(ds(e), 20); len(got) != 0 {
		t.Errorf("a card dated today was listed as a result: %d", len(got))
	}
}
