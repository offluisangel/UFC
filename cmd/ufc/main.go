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

func main() {
	args, err := tuiArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "uso: ufc [tui] [opciones]")
		os.Exit(2)
	}
	var (
		data    = flag.String("data", "data/events.json", "dataset JSON path")
		force   = flag.Bool("sync", false, "force a sync before starting")
		noSync  = flag.Bool("no-sync", false, "never sync automatically")
		verbose = flag.Bool("v", false, "verbose sync output")
	)
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "uso: ufc [tui] [opciones]")
		flag.PrintDefaults()
	}
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}

	ds, loadErr := dataset.Load(*data)

	if !*noSync {
		stale := loadErr != nil || time.Since(ds.SyncedAt) > staleAfter
		if *force || stale {
			fmt.Println("sincronizando desde ufcstats...")
			synced, err := fetch.Sync(fetch.Options{OutPath: *data, Verbose: *verbose})
			switch {
			case err != nil && loadErr != nil:
				fmt.Fprintln(os.Stderr, "falló la sincronización y no hay dataset cacheado:", err)
				os.Exit(1)
			case err != nil:
				fmt.Fprintln(os.Stderr, "falló la sincronización, uso el dataset cacheado:", err)
			default:
				ds, loadErr = synced, nil
				fmt.Printf("%d eventos sincronizados\n\n", ds.EventCount)
			}
		}
	}

	if _, err := tea.NewProgram(ui.New(ds, loadErr), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error de tui:", err)
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
	return nil, fmt.Errorf("comando desconocido %q", args[0])
}
