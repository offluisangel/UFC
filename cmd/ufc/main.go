package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/fetch"
	"github.com/offluisangel/UFC/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const staleAfter = 12 * time.Hour

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println("ufc", version)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "export" {
		os.Exit(runExport(os.Args[2:], os.Stdout, os.Stderr))
	}

	args, err := tuiArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Usage: ufc [tui] [options] | ufc export [options]")
		os.Exit(2)
	}
	var (
		data    = flag.String("data", "data/events.json", "dataset JSON path")
		force   = flag.Bool("sync", false, "force a sync before starting")
		noSync  = flag.Bool("no-sync", false, "never sync automatically")
		verbose = flag.Bool("v", false, "verbose sync output")
	)
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: ufc [tui] [options]")
		flag.PrintDefaults()
	}
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}

	ds, loadErr := dataset.Load(*data)

	if !*noSync {
		stale := loadErr != nil || time.Since(ds.SyncedAt) > staleAfter
		if *force || stale {
			fmt.Println("Syncing with ufcstats...")
			synced, err := fetch.Sync(fetch.Options{OutPath: *data, Verbose: *verbose})
			switch {
			case err != nil && loadErr != nil:
				fmt.Fprintln(os.Stderr, "Sync failed and no cached dataset is available:", err)
				os.Exit(1)
			case err != nil:
				fmt.Fprintln(os.Stderr, "Sync failed; using cached dataset:", err)
			default:
				ds, loadErr = synced, nil
				fmt.Printf("Synced %d events\n\n", ds.EventCount)
			}
		}
	}

	if _, err := tea.NewProgram(ui.New(ds, loadErr), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}
}

func tuiArgs(args []string) ([]string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args, nil
	}
	if args[0] == "tui" {
		return args[1:], nil
	}
	return nil, fmt.Errorf("unknown command %q", args[0])
}
