package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/export"
)

func runExport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	data := fs.String("data", "data/events.json", "dataset JSON path")
	out := fs.String("o", "", "output file; use - for stdout (default: a name based on the event)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: ufc export [options]")
		fmt.Fprintln(stderr, "Export the next event as an all-day .ics file")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ds, err := dataset.Load(*data)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	e, ok := dataset.Next(ds)
	if !ok {
		fmt.Fprintln(stderr, "No upcoming events to export")
		return 1
	}

	ics := export.ICS(e, time.Now())
	if *out == "-" {
		if _, err := stdout.Write(ics); err != nil {
			fmt.Fprintln(stderr, "Failed to write:", err)
			return 1
		}
		return 0
	}

	path := *out
	if path == "" {
		path = export.Filename(e)
	}
	if err := os.WriteFile(path, ics, 0o644); err != nil {
		fmt.Fprintln(stderr, "Failed to save file:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Saved %s (%s)\n", path, e.Name)
	return 0
}
