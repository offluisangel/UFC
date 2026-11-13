// Package apppaths provides platform-specific paths used by UFC commands.
package apppaths

import (
	"fmt"
	"os"
	"path/filepath"
)

// EventsCachePath returns the per-user UFC events cache path.
func EventsCachePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve UFC user cache directory: %w", err)
	}
	return filepath.Join(cacheDir, "ufc", "events.json"), nil
}

// DataPath uses an explicit path unchanged, or the per-user cache when omitted.
func DataPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	path, err := EventsCachePath()
	if err != nil {
		return "", fmt.Errorf("resolve default UFC dataset path: %w", err)
	}
	return path, nil
}
