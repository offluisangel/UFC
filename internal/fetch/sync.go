package fetch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/offluisangel/UFC/internal/model"
)

const DefaultRecentResults = 20

type Options struct {
	OutPath       string
	Verbose       bool
	RecentResults int
}

func (o Options) recentResultsLimit() int {
	if o.RecentResults == 0 {
		return DefaultRecentResults
	}
	return o.RecentResults
}

func Sync(opts Options) (model.Dataset, error) {
	client, err := NewClient()
	if err != nil {
		return model.Dataset{}, err
	}

	html, err := client.Upcoming()
	if err != nil {
		return model.Dataset{}, fmt.Errorf("fetch upcoming: %w", err)
	}
	if opts.Verbose {
		fmt.Println("upcoming: fetched")
	}

	links, err := parseEventLinks(html)
	if err != nil {
		return model.Dataset{}, err
	}
	if opts.Verbose {
		fmt.Printf("upcoming: %d events\n", len(links))
	}

	now := time.Now().UTC()
	ds := model.Dataset{
		Source:   "ufcstats.com",
		SyncedAt: now,
		Events:   make([]model.Event, 0, len(links)),
	}

	ds.Events = append(ds.Events, fetchEvents(client, links, model.StatusUpcoming, now, opts)...)

	if limit := opts.recentResultsLimit(); limit > 0 {
		past, err := fetchRecentResults(client, limit, now, opts)
		if err != nil {
			return model.Dataset{}, err
		}
		ds.Events = append(ds.Events, past...)
	}

	ds.Events, _ = dedupeByID(ds.Events, opts)

	ds.EventCount = len(ds.Events)
	if ds.EventCount == 0 {
		return ds, fmt.Errorf("sync: no events parsed, refusing to overwrite dataset")
	}

	totalFights := 0
	thin := make([]string, 0)
	for _, e := range ds.Events {
		totalFights += len(e.Card)
		if len(e.Card) == 0 {
			thin = append(thin, e.Name)
		}
	}

	if totalFights == 0 {
		return ds, fmt.Errorf(
			"sync: %d events parsed but 0 fights total, refusing to overwrite dataset (parser broken?)",
			ds.EventCount)
	}

	sort.SliceStable(ds.Events, func(i, j int) bool {
		return ds.Events[i].Date.Before(ds.Events[j].Date)
	})

	if err := writeDataset(ds, opts.OutPath); err != nil {
		return ds, err
	}
	if opts.Verbose {
		fmt.Printf("wrote %s (%d events, %d fights)\n", opts.OutPath, ds.EventCount, totalFights)
		if len(thin) > 0 {
			fmt.Printf("warning: %d event(s) with an empty card: %s\n",
				len(thin), strings.Join(thin, ", "))
		}
	}
	return ds, nil
}

func dedupeByID(events []model.Event, opts Options) ([]model.Event, int) {
	seen := make(map[string]bool, len(events))
	out := make([]model.Event, 0, len(events))
	dropped := 0
	for _, e := range events {
		if seen[e.ID] {
			dropped++
			if opts.Verbose {
				fmt.Printf("  dropped duplicate: %s (%s, marked %s)\n", e.Name, e.Date.Format("2006-01-02"), e.Status)
			}
			continue
		}
		seen[e.ID] = true
		out = append(out, e)
	}
	if dropped > 0 && opts.Verbose {
		fmt.Printf("dedupe: dropped %d event(s) already present\n", dropped)
	}
	return out, dropped
}

type htmlClient interface {
	Completed(page int) (string, error)
	EventDetail(id string) (string, error)
}

const completedPageCap = 10

func recentEventLinks(c htmlClient, limit int) ([]eventLink, error) {
	if limit <= 0 {
		return nil, nil
	}
	out := make([]eventLink, 0, limit)
	for page := 1; page <= completedPageCap && len(out) < limit; page++ {
		html, err := c.Completed(page)
		if err != nil {
			return nil, fmt.Errorf("fetch completed page %d: %w", page, err)
		}
		links, err := parseEventLinks(html)
		if err != nil {
			if len(out) > 0 {
				break
			}
			return nil, err
		}
		for _, l := range links {
			if len(out) == limit {
				break
			}
			out = append(out, l)
		}
	}
	return out, nil
}

func fetchRecentResults(c htmlClient, limit int, now time.Time, opts Options) ([]model.Event, error) {
	links, err := recentEventLinks(c, limit)
	if err != nil {
		if opts.Verbose {
			fmt.Printf("results: skipped (%v)\n", err)
		}
		return nil, nil
	}
	if opts.Verbose {
		fmt.Printf("results: %d recent events\n", len(links))
	}
	return fetchEvents(c, links, model.StatusCompleted, now, opts), nil
}

func fetchEvents(c htmlClient, links []eventLink, status string, now time.Time, opts Options) []model.Event {
	out := make([]model.Event, 0, len(links))
	for i, link := range links {
		detail, err := c.EventDetail(link.ID)
		if err != nil {
			if opts.Verbose {
				fmt.Printf("  [%d/%d] %s: FAILED %v\n", i+1, len(links), link.Name, err)
			}
			continue
		}
		ev, err := parseEventDetail(link.ID, detail)
		if err != nil {
			if opts.Verbose {
				fmt.Printf("  [%d/%d] %s: PARSE FAILED %v\n", i+1, len(links), link.Name, err)
			}
			continue
		}
		ev.Status = status
		ev.FetchedAt = now
		out = append(out, ev)
		if opts.Verbose {
			fmt.Printf("  [%d/%d] %s | %s | %d fights\n",
				i+1, len(links), ev.Name, ev.Date.Format("2006-01-02"), len(ev.Card))
		}
	}
	return out
}

func writeDataset(ds model.Dataset, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	buf, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}
