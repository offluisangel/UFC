package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/offluisangel/UFC/internal/dataset"
	"github.com/offluisangel/UFC/internal/model"
)

func TestExportKeyWritesTheNextEvent(t *testing.T) {
	dir := t.TempDir()
	ds := resultsDataset(t, 2)
	m := Model{dataset: ds, width: 80, height: 30, exportDir: dir}

	m = press(t, m, "e")

	files, err := filepath.Glob(filepath.Join(dir, "*.ics"))
	if err != nil || len(files) != 1 {
		t.Fatalf("want exactly one .ics in %s, got %v (err %v)", dir, files, err)
	}
	buf, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(buf), "SUMMARY:UFC Futura") {
		t.Errorf("exported the wrong event:\n%s", buf)
	}
	if strings.Contains(string(buf), "Pasado") {
		t.Errorf("exported more than the next event:\n%s", buf)
	}

	if !m.noticeOK || !strings.Contains(m.helpView(), "Saved") {
		t.Errorf("no confirmation shown: %q", stripANSI(m.helpView()))
	}

	m = press(t, m, "j")
	if m.notice != "" {
		t.Errorf("notice survived the next key press: %q", m.notice)
	}
}

func TestExportKeyWithoutUpcomingEvent(t *testing.T) {
	dir := t.TempDir()
	ds := model.Dataset{Events: []model.Event{{
		ID:     "old",
		Name:   "UFC Viejo",
		Date:   dataset.Today().Add(-72 * time.Hour),
		Status: model.StatusCompleted,
	}}}
	m := press(t, Model{dataset: ds, width: 80, height: 30, exportDir: dir}, "e")

	if m.noticeOK {
		t.Error("reported success with nothing to export")
	}
	if files, _ := filepath.Glob(filepath.Join(dir, "*.ics")); len(files) != 0 {
		t.Errorf("wrote %v with nothing to export", files)
	}
	if !strings.Contains(stripANSI(m.helpView()), "No upcoming event") {
		t.Errorf("no explanation shown: %q", stripANSI(m.helpView()))
	}
}

func TestExportKeyIsAdvertisedWhereItApplies(t *testing.T) {
	ds := resultsDataset(t, 2)
	for v, name := range viewNames {
		got := stripANSI(Model{dataset: ds, current: view(v), width: 100, height: 30}.helpView())
		want := name == "next" || name == "tonight" || name == "card"
		if has := strings.Contains(got, "e export"); has != want {
			t.Errorf("view %s: export hint present=%v, want %v: %q", name, has, want, got)
		}
	}
}
