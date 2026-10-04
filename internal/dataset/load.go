package dataset

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/offluisangel/UFC/internal/model"
)

func Load(path string) (model.Dataset, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.Dataset{}, fmt.Errorf("no hay dataset en %s (corré: go run ./cmd/ufc-sync)", path)
		}
		return model.Dataset{}, err
	}
	var ds model.Dataset
	if err := json.Unmarshal(buf, &ds); err != nil {
		return model.Dataset{}, fmt.Errorf("no se pudo leer %s: %w", path, err)
	}
	if len(ds.Events) == 0 {
		return model.Dataset{}, fmt.Errorf("el dataset %s no tiene eventos (corré: go run ./cmd/ufc-sync)", path)
	}
	return ds, nil
}

func DateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func Today() time.Time {
	return DateOnly(time.Now())
}

func Upcoming(ds model.Dataset) []model.Event {
	return upcomingAsOf(ds, Today())
}

func isFought(e model.Event, today time.Time) bool {
	return e.Status == model.StatusCompleted && DateOnly(e.Date).Before(today)
}

func upcomingAsOf(ds model.Dataset, today time.Time) []model.Event {
	out := make([]model.Event, 0, len(ds.Events))
	for _, e := range ds.Events {
		if isFought(e, today) {
			continue
		}
		if !DateOnly(e.Date).Before(today) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date.Before(out[j].Date)
	})
	return out
}

func RecentResults(ds model.Dataset, limit int) []model.Event {
	today := Today()
	out := make([]model.Event, 0, len(ds.Events))
	for _, e := range ds.Events {
		if !isFought(e, today) {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date.After(out[j].Date)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func Next(ds model.Dataset) (model.Event, bool) {
	return nextAsOf(ds, Today())
}

func nextAsOf(ds model.Dataset, today time.Time) (model.Event, bool) {
	up := upcomingAsOf(ds, today)
	if len(up) == 0 {
		return model.Event{}, false
	}
	return up[0], true
}

func Tonight(ds model.Dataset) (model.Event, bool) {
	return tonightAsOf(ds, Today())
}

func tonightAsOf(ds model.Dataset, today time.Time) (model.Event, bool) {
	for _, e := range ds.Events {
		if DateOnly(e.Date).Equal(DateOnly(today)) {
			return e, true
		}
	}
	return model.Event{}, false
}

func MainEvent(e model.Event) (model.Fight, bool) {
	for _, f := range e.Card {
		if f.IsMainEvent {
			return f, true
		}
	}
	return model.Fight{}, false
}
