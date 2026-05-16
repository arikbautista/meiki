package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupDataDir creates a temporary directory that acts as XDG_DATA_HOME
// for the duration of the test. Returns the meiki data dir path.
func setupDataDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	setEnv(t, "XDG_DATA_HOME", tmp)
	return filepath.Join(tmp, "meiki")
}

// --- LoadState tests ---

func TestLoadState_ZeroValueWhenFileAbsent(t *testing.T) {
	setupDataDir(t)

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v, want nil", err)
	}
	if s.LastBriefTS != "" {
		t.Errorf("LastBriefTS = %q, want empty string", s.LastBriefTS)
	}
	if s.LastReviewTS != "" {
		t.Errorf("LastReviewTS = %q, want empty string", s.LastReviewTS)
	}
}

func TestLoadState_ReadsExistingFile(t *testing.T) {
	dataDir := setupDataDir(t)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	content := `{"last_brief_ts":"2026-05-16T08:00:00Z","last_review_ts":"2026-05-15T23:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dataDir, "state.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if s.LastBriefTS != "2026-05-16T08:00:00Z" {
		t.Errorf("LastBriefTS = %q, want %q", s.LastBriefTS, "2026-05-16T08:00:00Z")
	}
	if s.LastReviewTS != "2026-05-15T23:00:00Z" {
		t.Errorf("LastReviewTS = %q, want %q", s.LastReviewTS, "2026-05-15T23:00:00Z")
	}
}

func TestLoadState_ErrorOnInvalidJSON(t *testing.T) {
	dataDir := setupDataDir(t)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dataDir, "state.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := LoadState()
	if err == nil {
		t.Error("LoadState() error = nil, want parse error for invalid JSON")
	}
}

// --- SaveState tests ---

func TestSaveState_WritesReadableJSON(t *testing.T) {
	setupDataDir(t)

	want := State{
		LastBriefTS:  "2026-05-16T09:00:00Z",
		LastReviewTS: "2026-05-15T22:00:00Z",
	}

	if err := SaveState(want); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	got, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() after SaveState error = %v", err)
	}
	if got.LastBriefTS != want.LastBriefTS {
		t.Errorf("LastBriefTS = %q, want %q", got.LastBriefTS, want.LastBriefTS)
	}
	if got.LastReviewTS != want.LastReviewTS {
		t.Errorf("LastReviewTS = %q, want %q", got.LastReviewTS, want.LastReviewTS)
	}
}

func TestSaveState_Atomic_NoTempFileLeftBehind(t *testing.T) {
	dataDir := setupDataDir(t)

	s := State{LastBriefTS: "2026-05-16T10:00:00Z"}
	if err := SaveState(s); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if name != "state.json" {
			t.Errorf("unexpected file left in data dir: %s", name)
		}
	}
}

func TestSaveState_CreatesDataDirIfAbsent(t *testing.T) {
	setupDataDir(t)
	// DataDir does not exist yet — SaveState must create it.
	s := State{LastBriefTS: "2026-05-16T11:00:00Z"}
	if err := SaveState(s); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	if _, err := LoadState(); err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
}

// --- UpdateBriefTS tests ---

func TestUpdateBriefTS_UpdatesOnlyBriefField(t *testing.T) {
	setupDataDir(t)

	initial := State{
		LastBriefTS:  "2026-05-15T08:00:00Z",
		LastReviewTS: "2026-05-15T23:00:00Z",
	}
	if err := SaveState(initial); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	now := time.Date(2026, 5, 16, 9, 30, 0, 0, time.UTC)
	if err := UpdateBriefTS(now); err != nil {
		t.Fatalf("UpdateBriefTS() error = %v", err)
	}

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	wantBrief := "2026-05-16T09:30:00Z"
	if s.LastBriefTS != wantBrief {
		t.Errorf("LastBriefTS = %q, want %q", s.LastBriefTS, wantBrief)
	}
	// LastReviewTS must be unchanged.
	if s.LastReviewTS != initial.LastReviewTS {
		t.Errorf("LastReviewTS = %q, want %q (unchanged)", s.LastReviewTS, initial.LastReviewTS)
	}
}

func TestUpdateBriefTS_NormalizesToUTC(t *testing.T) {
	setupDataDir(t)

	// Provide a non-UTC time; the stored value must be UTC.
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 5, 16, 17, 0, 0, 0, loc) // 17:00 UTC+8 == 09:00 UTC

	if err := UpdateBriefTS(now); err != nil {
		t.Fatalf("UpdateBriefTS() error = %v", err)
	}

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	wantBrief := "2026-05-16T09:00:00Z"
	if s.LastBriefTS != wantBrief {
		t.Errorf("LastBriefTS = %q, want %q", s.LastBriefTS, wantBrief)
	}
}

// --- UpdateReviewTS tests ---

func TestUpdateReviewTS_UpdatesOnlyReviewField(t *testing.T) {
	setupDataDir(t)

	initial := State{
		LastBriefTS:  "2026-05-15T08:00:00Z",
		LastReviewTS: "2026-05-15T23:00:00Z",
	}
	if err := SaveState(initial); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	now := time.Date(2026, 5, 16, 22, 45, 0, 0, time.UTC)
	if err := UpdateReviewTS(now); err != nil {
		t.Fatalf("UpdateReviewTS() error = %v", err)
	}

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	wantReview := "2026-05-16T22:45:00Z"
	if s.LastReviewTS != wantReview {
		t.Errorf("LastReviewTS = %q, want %q", s.LastReviewTS, wantReview)
	}
	// LastBriefTS must be unchanged.
	if s.LastBriefTS != initial.LastBriefTS {
		t.Errorf("LastBriefTS = %q, want %q (unchanged)", s.LastBriefTS, initial.LastBriefTS)
	}
}

func TestUpdateReviewTS_NormalizesToUTC(t *testing.T) {
	setupDataDir(t)

	loc := time.FixedZone("UTC-5", -5*60*60)
	now := time.Date(2026, 5, 16, 18, 0, 0, 0, loc) // 18:00 UTC-5 == 23:00 UTC

	if err := UpdateReviewTS(now); err != nil {
		t.Fatalf("UpdateReviewTS() error = %v", err)
	}

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	wantReview := "2026-05-16T23:00:00Z"
	if s.LastReviewTS != wantReview {
		t.Errorf("LastReviewTS = %q, want %q", s.LastReviewTS, wantReview)
	}
}
