package export

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/offluisangel/UFC/internal/model"
)

const (
	dateLayout  = "20060102"
	stampLayout = "20060102T150405Z"

	foldLimit = 75

	uidDomain = "ufc-tui"
)

func ICS(e model.Event, now time.Time) []byte {
	start := dayOf(e.Date)
	end := start.AddDate(0, 0, 1) // DTEND is exclusive for all-day events

	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//ufc-tui//fight nights//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + uid(e),
		"DTSTAMP:" + now.UTC().Format(stampLayout),
		"DTSTART;VALUE=DATE:" + start.Format(dateLayout),
		"DTEND;VALUE=DATE:" + end.Format(dateLayout),
		"SUMMARY:" + escape(e.Name),
	}
	if e.Location != "" {
		lines = append(lines, "LOCATION:"+escape(e.Location))
	}
	lines = append(lines,
		"DESCRIPTION:"+escape(description(e)),
		"TRANSP:TRANSPARENT",
		"END:VEVENT",
		"END:VCALENDAR",
	)

	var b strings.Builder
	for _, l := range lines {
		b.WriteString(fold(l))
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func Filename(e model.Event) string {
	slug := slugify(e.Name)
	if slug == "" {
		slug = "ufc-event"
	}
	return slug + "-" + dayOf(e.Date).Format("2006-01-02") + ".ics"
}

func description(e model.Event) string {
	var parts []string
	for _, f := range e.Card {
		if !f.IsMainEvent || len(f.Fighters) < 2 {
			continue
		}
		line := "Main event: " + f.Fighters[0] + " vs " + f.Fighters[1]
		if f.WeightClass != "" {
			line += " (" + f.WeightClass + ")"
		}
		parts = append(parts, line)
		break
	}
	switch n := len(e.Card); n {
	case 0:
		parts = append(parts, "Fight card not announced.")
	case 1:
		parts = append(parts, "1 fight announced.")
	default:
		parts = append(parts, fmt.Sprintf("%d fights announced.", n))
	}
	parts = append(parts, "Event time not included.")
	return strings.Join(parts, "\n")
}

func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func uid(e model.Event) string {
	id := e.ID
	if id == "" {
		id = slugify(e.Name) + "-" + dayOf(e.Date).Format(dateLayout)
	}
	return id + "@" + uidDomain
}

var textEscaper = strings.NewReplacer(
	`\`, `\\`,
	`;`, `\;`,
	`,`, `\,`,
	"\r\n", `\n`,
	"\n", `\n`,
	"\r", "",
)

func escape(s string) string {
	return textEscaper.Replace(s)
}

func fold(line string) string {
	if len(line) <= foldLimit {
		return line
	}
	var b strings.Builder
	n := 0
	for _, r := range line {
		size := utf8.RuneLen(r)
		if n+size > foldLimit {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	return b.String()
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
