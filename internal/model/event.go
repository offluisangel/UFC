package model

import "time"

type Fight struct {
	Fighters    []string `json:"fighters"`
	FighterIDs  []string `json:"fighterIds,omitempty"`
	WeightClass string   `json:"weightClass"`
	Method      string   `json:"method,omitempty"`
	Round       string   `json:"round,omitempty"`
	Time        string   `json:"time,omitempty"`
	Winner      string   `json:"winner,omitempty"`
	IsMainEvent bool     `json:"isMainEvent"`
}

const (
	StatusUpcoming  = "upcoming"
	StatusCompleted = "completed"
)

type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Date      time.Time `json:"date"`
	Location  string    `json:"location"`
	Status    string    `json:"status"`
	Card      []Fight   `json:"card,omitempty"`
	FetchedAt time.Time `json:"fetchedAt"`
}

type Dataset struct {
	Source     string    `json:"source"`
	SyncedAt   time.Time `json:"syncedAt"`
	EventCount int       `json:"eventCount"`
	Events     []Event   `json:"events"`
}

func ParseEventDate(s string) (time.Time, error) {
	layouts := []string{
		"January 02, 2006",
		"Jan 02, 2006",
		"January 2, 2006",
		"Jan 2, 2006",
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}
