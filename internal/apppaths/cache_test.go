package apppaths

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestEventsCachePathUsesOSUserCacheDirectory(t *testing.T) {
	cacheRoot := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", cacheRoot)
	} else {
		t.Setenv("XDG_CACHE_HOME", cacheRoot)
	}

	got, err := EventsCachePath()
	if err != nil {
		t.Fatalf("EventsCachePath() error = %v", err)
	}
	want := filepath.Join(cacheRoot, "ufc", "events.json")
	if got != want {
		t.Fatalf("EventsCachePath() = %q, want %q", got, want)
	}
}

func TestDataPathPreservesExplicitPath(t *testing.T) {
	const explicit = "../existing/events.json"
	got, err := DataPath(explicit)
	if err != nil {
		t.Fatalf("DataPath(%q) error = %v", explicit, err)
	}
	if got != explicit {
		t.Fatalf("DataPath(%q) = %q, want exact explicit path", explicit, got)
	}
}
