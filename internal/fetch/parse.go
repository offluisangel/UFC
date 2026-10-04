package fetch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/offluisangel/UFC/internal/model"
)

var (
	eventIDRe     = regexp.MustCompile(`event-details/([a-z0-9]+)`)
	fighterHrefRe = regexp.MustCompile(`fighter-details/([a-z0-9]+)`)
	spaceRe       = regexp.MustCompile(`\s+`)
)

func clean(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

type eventLink struct {
	ID   string
	Name string
}

func parseEventLinks(html string) ([]eventLink, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse upcoming: %w", err)
	}

	seen := map[string]bool{}
	var out []eventLink

	doc.Find("a[href*='event-details/']").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := eventIDRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id := m[1]
		if seen[id] {
			return
		}
		name := clean(a.Text())
		if name == "" {
			return
		}
		seen[id] = true
		out = append(out, eventLink{ID: id, Name: name})
	})

	if len(out) == 0 {
		return nil, fmt.Errorf("parse upcoming: no event links found")
	}
	return out, nil
}

func labelValue(doc *goquery.Document, label string) string {
	var found string
	doc.Find("li.b-list__box-list-item").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		title := clean(li.Find("i.b-list__box-item-title").Text())
		if !strings.EqualFold(title, label) {
			return true
		}
		found = strings.TrimSpace(strings.TrimPrefix(clean(li.Text()), title))
		return false
	})
	return found
}

func parseEventDetail(id, html string) (model.Event, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return model.Event{}, fmt.Errorf("parse event %s: %w", id, err)
	}

	name := clean(doc.Find("h2.b-content__title").First().Text())
	if name == "" {
		name = clean(doc.Find("h1").First().Text())
	}
	if name == "" {
		return model.Event{}, fmt.Errorf("parse event %s: no title", id)
	}

	dateStr := labelValue(doc, "Date:")
	if dateStr == "" {
		dateStr = labelValue(doc, "Date")
	}
	if dateStr == "" {
		return model.Event{}, fmt.Errorf("parse event %s: no date", id)
	}
	date, err := model.ParseEventDate(dateStr)
	if err != nil {
		return model.Event{}, fmt.Errorf("parse event %s: date %q: %w", id, dateStr, err)
	}

	location := labelValue(doc, "Location:")
	if location == "" {
		location = labelValue(doc, "Location")
	}

	if doc.Find("table.b-fight-details__table").Length() == 0 {
		return model.Event{}, fmt.Errorf(
			"parse event %s: fight table not found in HTML (ufcstats markup changed?)", id)
	}

	card := parseCard(doc)
	if len(card) == 0 {
		if n := doc.Find("table tbody tr").Length(); n > 0 {
			return model.Event{}, fmt.Errorf(
				"parse event %s: %d fight row(s) in the table but none produced a fight (selectors changed?)", id, n)
		}
	}

	return model.Event{
		ID:       id,
		Name:     name,
		Date:     date,
		Location: location,
		Card:     card,
	}, nil
}

func parseCard(doc *goquery.Document) []model.Fight {
	var fights []model.Fight

	doc.Find("table tbody tr").Each(func(i int, tr *goquery.Selection) {
		cells := tr.Find("td")
		if cells.Length() < 2 {
			return
		}

		var fighters []string
		var fighterIDs []string
		tr.Find("a").Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			m := fighterHrefRe.FindStringSubmatch(href)
			if m == nil {
				return
			}
			if n := clean(a.Text()); n != "" {
				fighters = append(fighters, n)
				fighterIDs = append(fighterIDs, m[1])
			}
		})
		if len(fighters) == 0 {
			return
		}

		weight := ""
		method := ""
		round := ""
		fightTime := ""

		n := cells.Length()
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			txt := clean(td.Text())
			if txt == "" {
				return
			}
			switch j {
			case n - 4:
				weight = txt
			case n - 3:
				method = txt
			case n - 2:
				round = txt
			case n - 1:
				fightTime = txt
			}
		})

		fights = append(fights, model.Fight{
			Fighters:    fighters,
			FighterIDs:  fighterIDs,
			WeightClass: weight,
			Method:      method,
			Round:       round,
			Time:        fightTime,
			Winner:      winnerOf(cells, fighters),
			IsMainEvent: i == 0,
		})
	})

	return fights
}

func winnerOf(cells *goquery.Selection, fighters []string) string {
	if len(fighters) < 2 {
		return ""
	}
	flag := strings.ToLower(clean(cells.First().Find("i.b-flag__text").First().Text()))
	switch flag {
	case "win":
		return fighters[0]
	case "loss":
		return fighters[1]
	}
	return ""
}
