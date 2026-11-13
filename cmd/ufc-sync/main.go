package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/offluisangel/UFC/internal/apppaths"
	"github.com/offluisangel/UFC/internal/fetch"
)

func main() {
	var (
		out     = flag.String("out", "", "output JSON path (default: per-user cache)")
		verbose = flag.Bool("v", true, "verbose output")
	)
	flag.Parse()

	outPath, err := apppaths.DataPath(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ds, err := fetch.Sync(fetch.Options{OutPath: outPath, Verbose: *verbose})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Sync failed:", err)
		os.Exit(1)
	}
	if !*verbose {
		fmt.Printf("Synced %d events -> %s\n", ds.EventCount, outPath)
	}
}
