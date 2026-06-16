package showtimes

import (
	"os"
	"testing"
	"time"
)

func TestLoadStateMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())

	state := LoadState()

	if state.Status != "not-found" {
		t.Errorf("Status = %q, want %q", state.Status, "not-found")
	}
	if !state.LastNotFoundTs.Equal(epoch) {
		t.Errorf("LastNotFoundTs = %v, want %v", state.LastNotFoundTs, epoch)
	}
}

func TestLoadStateMissingTimestampField(t *testing.T) {
	t.Chdir(t.TempDir())

	// No last_not_found_ts key at all, as a hand-edited or older state.json
	// might have.
	if err := os.WriteFile(stateFile, []byte(`{"status":"found"}`), 0644); err != nil {
		t.Fatal(err)
	}

	state := LoadState()

	if state.Status != "found" {
		t.Errorf("Status = %q, want %q", state.Status, "found")
	}
	if !state.LastNotFoundTs.Equal(epoch) {
		t.Errorf("LastNotFoundTs = %v, want %v (zero value should normalize to epoch)", state.LastNotFoundTs, epoch)
	}
}

func TestSaveAndLoadStateRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	want := State{
		Status:            "found",
		LastNotFoundTs:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		CachedAPIKeyChunk: "/_next/static/chunks/abc.js",
	}
	SaveState(want)

	got := LoadState()

	if got.Status != want.Status {
		t.Errorf("Status = %q, want %q", got.Status, want.Status)
	}
	if !got.LastNotFoundTs.Equal(want.LastNotFoundTs) {
		t.Errorf("LastNotFoundTs = %v, want %v", got.LastNotFoundTs, want.LastNotFoundTs)
	}
	if got.CachedAPIKeyChunk != want.CachedAPIKeyChunk {
		t.Errorf("CachedAPIKeyChunk = %q, want %q", got.CachedAPIKeyChunk, want.CachedAPIKeyChunk)
	}
}
