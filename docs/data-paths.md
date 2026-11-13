# UFC data paths

UFC stores its events dataset in the operating system's per-user cache directory by default, at `ufc/events.json` beneath that directory. The path is resolved at runtime for the current user, so running `ufc`, `ufc export`, or `ufc-sync` from different working directories uses the same cache.

- `ufc [tui] --data PATH` selects a dataset for the TUI and any automatic sync.
- `ufc export --data PATH` selects the dataset to export.
- `ufc-sync --out PATH` selects where a sync writes its dataset.

Explicit paths are used as supplied; relative explicit paths remain relative to the current working directory. Calendar export destinations (`ufc export -o PATH`) also remain relative to the current working directory.

Existing datasets in the previous working-directory-relative `data/events.json` location are not moved or rewritten automatically. To continue using one, pass its path explicitly, for example `ufc --data data/events.json`, `ufc export --data data/events.json`, or `ufc-sync --out data/events.json`.
