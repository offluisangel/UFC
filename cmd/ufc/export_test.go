package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/offluisangel/UFC/internal/model"
)

func writeDataset(t *testing.T, path, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	ds := model.Dataset{Events: []model.Event{{Name: name, Date: time.Now().AddDate(1, 0, 0), Status: model.StatusUpcoming}}}
	data, err := json.Marshal(ds)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExportLoadsDefaultCacheFromAnotherWorkingDirectory(t *testing.T) {
	cache := t.TempDir()
	work := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", cache)
	} else {
		t.Setenv("XDG_CACHE_HOME", cache)
	}
	writeDataset(t, filepath.Join(cache, "ufc", "events.json"), "Cached Event")
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runExport([]string{"-o", "-"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "SUMMARY:Cached Event") {
		t.Fatalf("runExport() = (%d, stdout %q, stderr %q), want successful cached export", code, stdout.String(), stderr.String())
	}
}

func TestExportHonorsExplicitDataPath(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "chosen.json")
	writeDataset(t, dataPath, "Explicit Event")
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", t.TempDir())
	} else {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
	}

	var stdout, stderr bytes.Buffer
	code := runExport([]string{"--data", dataPath, "-o", "-"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "SUMMARY:Explicit Event") {
		t.Fatalf("runExport() = (%d, stdout %q, stderr %q), want successful explicit-path export", code, stdout.String(), stderr.String())
	}
}
