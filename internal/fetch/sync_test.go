package fetch

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/offluisangel/UFC/internal/model"
)

type fakeSite struct {
	pages   map[int]string
	details map[string]string
	calls   []string
}

func (f *fakeSite) Completed(page int) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("page=%d", page))
	html, ok := f.pages[page]
	if !ok {
		return "", fmt.Errorf("no page %d", page)
	}
	return html, nil
}

func (f *fakeSite) EventDetail(id string) (string, error) {
	f.calls = append(f.calls, "detail:"+id)
	html, ok := f.details[id]
	if !ok {
		return "", fmt.Errorf("no detail %s", id)
	}
	return html, nil
}

func (f *fakeSite) pageCalls() []string {
	var out []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, "page=") {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeSite) detailCount() int {
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, "detail:") {
			n++
		}
	}
	return n
}

func listing(ids ...string) string {
	var b strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&b, `<tr class="b-list__box-list-item">
			<td class="b-list__box-item">
				<a href="/event-details/%s">UFC %s</a>
			</td>
		</tr>`, id, id)
	}
	return `<html><body><table class="b-list__box">` + b.String() + `</table></body></html>`
}

func TestRecentEventLinksStopsAtTheLimit(t *testing.T) {

	site := &fakeSite{pages: map[int]string{1: listing("a", "b", "c", "d")}}

	links, err := recentEventLinks(site, 2)
	if err != nil {
		t.Fatalf("recentEventLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2", len(links))
	}
	if links[0].ID != "a" {
		t.Errorf("first link = %q, want a: the newest must come first", links[0].ID)
	}
	if got := site.pageCalls(); len(got) != 1 {
		t.Errorf("pages requested = %v, want exactly one", got)
	}
}

func TestRecentEventLinksWalksToTheNextPageWhenShort(t *testing.T) {
	site := &fakeSite{pages: map[int]string{
		1: listing("a", "b"),
		2: listing("c", "d"),
	}}

	links, err := recentEventLinks(site, 3)
	if err != nil {
		t.Fatalf("recentEventLinks: %v", err)
	}
	if len(links) != 3 {
		t.Fatalf("got %d links, want 3", len(links))
	}
	want := []string{"a", "b", "c"}
	for i, id := range want {
		if links[i].ID != id {
			t.Errorf("link %d = %q, want %q", i, links[i].ID, id)
		}
	}
	if got := site.pageCalls(); len(got) != 2 {
		t.Errorf("pages requested = %v, want two", got)
	}
}

func TestRecentEventLinksEndsOnAnEmptyPage(t *testing.T) {

	site := &fakeSite{pages: map[int]string{
		1: listing("a", "b"),
		2: `<html><body>nothing</body></html>`,
	}}

	links, err := recentEventLinks(site, 10)
	if err != nil {
		t.Fatalf("an exhausted history must not be an error: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("got %d links, want the 2 that existed", len(links))
	}
}

func TestRecentEventLinksFailsWhenTheFirstPageIsUnusable(t *testing.T) {

	site := &fakeSite{pages: map[int]string{1: `<html><body>nothing</body></html>`}}

	if _, err := recentEventLinks(site, 10); err == nil {
		t.Fatal("expected an error when the first page has no links")
	}
}

func TestRecentEventLinksStopsAtThePageCap(t *testing.T) {

	pages := map[int]string{}
	for p := 1; p <= completedPageCap+10; p++ {
		pages[p] = listing(fmt.Sprintf("p%d", p))
	}
	site := &fakeSite{pages: pages}

	links, err := recentEventLinks(site, 1000)
	if err != nil {
		t.Fatalf("recentEventLinks: %v", err)
	}
	if got := len(site.pageCalls()); got != completedPageCap {
		t.Errorf("pages requested = %d, want the cap of %d", got, completedPageCap)
	}
	if len(links) != completedPageCap {
		t.Errorf("got %d links, want %d", len(links), completedPageCap)
	}
}

func TestRecentEventLinksZeroLimitSkipsEverything(t *testing.T) {
	site := &fakeSite{pages: map[int]string{1: listing("a")}}

	links, err := recentEventLinks(site, 0)
	if err != nil {
		t.Fatalf("recentEventLinks: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("got %d links, want none", len(links))
	}
	if len(site.pageCalls()) != 0 {
		t.Errorf("a zero limit must not touch the network, got %v", site.pageCalls())
	}
}

func TestFetchEventsMarksStatus(t *testing.T) {
	detail := foughtCard(fightRow("win", "Alpha", "Bravo"))
	site := &fakeSite{details: map[string]string{"x": detail}}

	got := fetchEvents(site, []eventLink{{ID: "x", Name: "UFC 1"}}, model.StatusCompleted, time.Now(), Options{})
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Status != model.StatusCompleted {
		t.Errorf("status = %q, want %q", got[0].Status, model.StatusCompleted)
	}
	if got[0].Card[0].Winner != "Alpha" {
		t.Errorf("winner = %q, want Alpha", got[0].Card[0].Winner)
	}
}

func TestFetchEventsSkipsWhatItCannotRead(t *testing.T) {
	site := &fakeSite{details: map[string]string{
		"good": foughtCard(fightRow("win", "Alpha", "Bravo")),
	}}

	links := []eventLink{
		{ID: "bad-fetch", Name: "A"},
		{ID: "bad-html", Name: "B"},
		{ID: "good", Name: "C"},
	}
	got := fetchEvents(site, links, model.StatusCompleted, time.Now(), Options{})
	if len(got) != 1 {
		t.Fatalf("got %d events, want only the readable one", len(got))
	}
	if got[0].ID != "good" {
		t.Errorf("kept %q, want good", got[0].ID)
	}
}

func TestFetchRecentResultsSurvivesAFailedListing(t *testing.T) {
	site := &fakeSite{pages: map[int]string{}}
	got, err := fetchRecentResults(site, 5, time.Now(), Options{})
	if err != nil {
		t.Fatalf("fetchRecentResults must degrade, not fail: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d events, want none", len(got))
	}
}

func TestRecentResultsLimitOption(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{0, DefaultRecentResults},
		{5, 5},
		{-1, -1},
	}
	for _, tc := range tests {
		if got := (Options{RecentResults: tc.in}).recentResultsLimit(); got != tc.want {
			t.Errorf("RecentResults %d -> %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestDedupeByIDKeepsTheFirstCopy(t *testing.T) {
	upcoming := model.Event{ID: "x", Name: "UFC 332", Status: model.StatusUpcoming}
	future := model.Event{ID: "x", Name: "UFC 332", Status: model.StatusCompleted}
	other := model.Event{ID: "y", Name: "UFC 331", Status: model.StatusCompleted}

	got, dropped := dedupeByID([]model.Event{upcoming, future, other}, Options{})

	if dropped != 1 {
		t.Errorf("dropped %d, want 1", dropped)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].ID != "x" || got[0].Status != model.StatusUpcoming {
		t.Errorf("kept %q with status %q, want x as upcoming", got[0].ID, got[0].Status)
	}
	if got[1].ID != "y" {
		t.Errorf("second event = %q, want y", got[1].ID)
	}
}

func TestDedupeByIDOnAListWithoutRepeats(t *testing.T) {
	events := []model.Event{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	got, dropped := dedupeByID(events, Options{})
	if dropped != 0 {
		t.Errorf("dropped %d, want 0", dropped)
	}
	if len(got) != 3 {
		t.Errorf("got %d events, want 3", len(got))
	}
}
