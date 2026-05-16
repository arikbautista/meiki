package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/review"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runReview executes newReviewCmd() with the given arguments using the provided
// tmpDir as XDG_DATA_HOME so all file I/O goes to a temp directory.
// Returns stdout output and any error.
func runReview(t *testing.T, tmpDir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	cmd := newReviewCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return strings.TrimSpace(out.String()), err
}

// writeReviewEntry writes a single entry to the JSONL file for the given date
// in XDG_DATA_HOME/meiki/entries/YYYY/MM/YYYY-MM-DD.jsonl.
func writeReviewEntry(t *testing.T, tmpDir string, date time.Time, e entry.Entry) {
	t.Helper()
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	path := filepath.Join(tmpDir, "meiki", "entries", y, m, d+".jsonl")
	if _, err := entry.AppendEntryToPath(&e, path); err != nil {
		t.Fatalf("writeReviewEntry: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Basic operation — no entries
// ---------------------------------------------------------------------------

func TestReview_noEntries(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "# Daily Review") {
		t.Errorf("expected '# Daily Review' in stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "No entries recorded today.") {
		t.Errorf("expected 'No entries recorded today.' in stdout, got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Review file is written to the correct path
// ---------------------------------------------------------------------------

func TestReview_writesFileToCorrectPath(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	_, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The review file should exist at the expected path.
	dataDir := filepath.Join(tmpDir, "meiki")
	reviewPath := review.ReviewFilePath(dataDir, now)
	if _, err := os.Stat(reviewPath); os.IsNotExist(err) {
		t.Errorf("expected review file at %s, but it does not exist", reviewPath)
	}
}

// ---------------------------------------------------------------------------
// Review file path is printed to stdout
// ---------------------------------------------------------------------------

func TestReview_printsPathToStdout(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	stdout, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// stdout should contain the review file path.
	dataDir := filepath.Join(tmpDir, "meiki")
	reviewPath := review.ReviewFilePath(dataDir, now)
	if !strings.Contains(stdout, reviewPath) {
		t.Errorf("expected review path %q in stdout, got:\n%s", reviewPath, stdout)
	}
}

// ---------------------------------------------------------------------------
// Review content appears in stdout
// ---------------------------------------------------------------------------

func TestReview_contentInStdout(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACH001", "achievement", "Shipped the capture command", "meiki", now)
	writeReviewEntry(t, tmpDir, now, e)

	stdout, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Shipped the capture command") {
		t.Errorf("expected achievement content in stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "## Achievements") {
		t.Errorf("expected '## Achievements' section in stdout:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// --silent suppresses stdout but still writes the file
// ---------------------------------------------------------------------------

func TestReview_silentFlag_suppressesOutput(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACH001", "achievement", "Shipped something", "meiki", now)
	writeReviewEntry(t, tmpDir, now, e)

	stdout, err := runReview(t, tmpDir, "--silent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "" {
		t.Errorf("expected empty stdout with --silent, got:\n%s", stdout)
	}
}

func TestReview_silentFlag_stillWritesFile(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	_, err := runReview(t, tmpDir, "--silent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dataDir := filepath.Join(tmpDir, "meiki")
	reviewPath := review.ReviewFilePath(dataDir, now)
	if _, err := os.Stat(reviewPath); os.IsNotExist(err) {
		t.Errorf("expected review file at %s with --silent, but it does not exist", reviewPath)
	}
}

// ---------------------------------------------------------------------------
// Updates last_review_ts in state.json
// ---------------------------------------------------------------------------

func TestReview_updatesLastReviewTS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	_, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := config.LoadState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.LastReviewTS == "" {
		t.Error("expected last_review_ts to be set in state.json, got empty string")
	}
}

// ---------------------------------------------------------------------------
// Idempotency — running twice produces the same file
// ---------------------------------------------------------------------------

func TestReview_idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	e := makeEntry("ACH001", "achievement", "Something great", "meiki", now)
	writeReviewEntry(t, tmpDir, now, e)

	_, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}

	dataDir := filepath.Join(tmpDir, "meiki")
	reviewPath := review.ReviewFilePath(dataDir, now)
	content1, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("read file after first run: %v", err)
	}

	_, err = runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}

	content2, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("read file after second run: %v", err)
	}

	if string(content1) != string(content2) {
		t.Errorf("review is not idempotent:\nfirst run:\n%s\nsecond run:\n%s", content1, content2)
	}
}

// ---------------------------------------------------------------------------
// Creates intermediate year/month directories
// ---------------------------------------------------------------------------

func TestReview_createsIntermediateDirs(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now().UTC()

	_, err := runReview(t, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The year/month directory under reviews/ must exist.
	dataDir := filepath.Join(tmpDir, "meiki")
	monthDir := filepath.Join(dataDir, "reviews", now.Format("2006"), now.Format("01"))
	if _, err := os.Stat(monthDir); os.IsNotExist(err) {
		t.Errorf("expected reviews year/month directory at %s, but it does not exist", monthDir)
	}
}

// ---------------------------------------------------------------------------
// Exit code is 0
// ---------------------------------------------------------------------------

func TestReview_exitCodeZero(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := runReview(t, tmpDir)
	if err != nil {
		t.Errorf("expected nil error (exit 0), got %v", err)
	}
}
