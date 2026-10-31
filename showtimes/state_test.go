package showtimes

import (
	"os"
	"testing"
	"time"
)

func TestLoadState(t *testing.T) {
	fresh := State{Status: statusNotFound}

	tests := []struct {
		name string
		file string // "" means no state file
		want State
	}{
		{name: "missing file", want: fresh},
		{name: "invalid JSON", file: `{"status":`, want: fresh},
		{
			// A state file from an older version, with an extra field.
			name: "older file",
			file: `{"status":"found","last_not_found_ts":"1970-01-01T00:00:00Z","cached_api_key_chunk":"/x.js"}`,
			want: State{Status: statusFound, LastNotFoundTs: time.Unix(0, 0).UTC()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tt.file != "" {
				if err := os.WriteFile(stateFile, []byte(tt.file), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got := loadState()
			if got.Status != tt.want.Status || !got.LastNotFoundTs.Equal(tt.want.LastNotFoundTs) {
				t.Errorf("loadState() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSaveStateRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	want := State{Status: statusNotFound, LastNotFoundTs: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	if err := saveState(want); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	got := loadState()
	if got.Status != want.Status || !got.LastNotFoundTs.Equal(want.LastNotFoundTs) {
		t.Errorf("loadState() = %+v, want %+v", got, want)
	}
}
