package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State tracks timestamps used for debouncing the brief and review commands.
type State struct {
	LastBriefTS  string `json:"last_brief_ts"`  // ISO-8601 UTC
	LastReviewTS string `json:"last_review_ts"` // ISO-8601 UTC
}

// statePath returns the full path to the state.json file.
func statePath() string {
	return filepath.Join(DataDir(), "state.json")
}

// LoadState reads state.json from the data directory.
// If the file does not exist, a zero-value State is returned without error.
func LoadState() (State, error) {
	var s State
	data, err := os.ReadFile(statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse state.json: %w", err)
	}
	return s, nil
}

// SaveState atomically writes state to state.json.
// It writes to a temp file in the same directory and renames it into place.
func SaveState(s State) error {
	dir := DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "state-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, statePath()); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// UpdateBriefTS loads the current state, updates last_brief_ts to now, and saves.
func UpdateBriefTS(now time.Time) error {
	s, err := LoadState()
	if err != nil {
		return err
	}
	s.LastBriefTS = now.UTC().Format(time.RFC3339)
	return SaveState(s)
}

// UpdateReviewTS loads the current state, updates last_review_ts to now, and saves.
func UpdateReviewTS(now time.Time) error {
	s, err := LoadState()
	if err != nil {
		return err
	}
	s.LastReviewTS = now.UTC().Format(time.RFC3339)
	return SaveState(s)
}
