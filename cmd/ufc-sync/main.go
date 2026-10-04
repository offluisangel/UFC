package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/offluisangel/UFC/internal/fetch"
)

func main() {
	var (
		out     = flag.String("out", "data/events.json", "output JSON path")
		verbose = flag.Bool("v", true, "verbose output")
	)
	flag.Parse()

	ds, err := fetch.Sync(fetch.Options{OutPath: *out, Verbose: *verbose})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync failed:", err)
		os.Exit(1)
	}
	if !*verbose {
		fmt.Printf("synced %d events -> %s\n", ds.EventCount, *out)
	}
}
