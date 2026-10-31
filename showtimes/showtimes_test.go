package showtimes

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestDecide(t *testing.T) {
	const pageURL = "https://uae.voxcinemas.com/movies/backrooms"
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		state     State
		found     bool
		wantTitle string // "" means no notification
		want      State
	}{
		{
			name:      "showtimes appear",
			state:     State{Status: statusNotFound},
			found:     true,
			wantTitle: "Showtimes available",
			want:      State{Status: statusFound},
		},
		{
			name:  "showtimes still there",
			state: State{Status: statusFound},
			found: true,
			want:  State{Status: statusFound},
		},
		{
			name:      "first check, no showtimes",
			state:     State{Status: statusNotFound},
			wantTitle: "No showtimes yet",
			want:      State{Status: statusNotFound, LastNotFoundTs: now},
		},
		{
			name:      "showtimes gone, interval passed",
			state:     State{Status: statusFound, LastNotFoundTs: now.Add(-NotFoundInterval)},
			wantTitle: "No showtimes yet",
			want:      State{Status: statusNotFound, LastNotFoundTs: now},
		},
		{
			name:  "no showtimes, interval not passed",
			state: State{Status: statusNotFound, LastNotFoundTs: now.Add(-time.Minute)},
			want:  State{Status: statusNotFound, LastNotFoundTs: now.Add(-time.Minute)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n := decide(tt.state, tt.found, now, pageURL)

			gotTitle := ""
			if n != nil {
				gotTitle = n.Title
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("notification title = %q, want %q", gotTitle, tt.wantTitle)
			}
			if got.Status != tt.want.Status || !got.LastNotFoundTs.Equal(tt.want.LastNotFoundTs) {
				t.Errorf("state = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Regression: a second run must not send the "available" notification again.
func TestCheckNotifiesOnceWhenShowtimesAppear(t *testing.T) {
	t.Chdir(t.TempDir())
	sent := fakeNtfy(t, http.StatusOK)
	pageURL := servePage(t, http.StatusOK, showingPage)

	for run := 1; run <= 2; run++ {
		if err := Check(pageURL, "topic"); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}

	if len(*sent) != 1 {
		t.Fatalf("sent %d notifications, want 1", len(*sent))
	}
	if got := loadState().Status; got != statusFound {
		t.Errorf("saved status = %q, want %q", got, statusFound)
	}
}

// Regression: a failed page check must fail the run (exit code 1 in main)
// and must not change the state.
func TestCheckPageErrorKeepsState(t *testing.T) {
	t.Chdir(t.TempDir())
	sent := fakeNtfy(t, http.StatusOK)

	if err := Check(servePage(t, http.StatusNotFound, "not found"), "topic"); err == nil {
		t.Fatal("Check returned no error for a 404 page")
	}
	if len(*sent) != 0 {
		t.Errorf("sent %d notifications, want 0", len(*sent))
	}
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Errorf("state file was written after a failed check (stat err = %v)", err)
	}
}

// Regression: if ntfy.sh fails, the state must not change to "found", so the
// next run sends the notification again.
func TestCheckNotificationErrorRetriesNextRun(t *testing.T) {
	t.Chdir(t.TempDir())
	pageURL := servePage(t, http.StatusOK, showingPage)

	fakeNtfy(t, http.StatusInternalServerError)
	if err := Check(pageURL, "topic"); err == nil {
		t.Fatal("Check returned no error when ntfy.sh failed")
	}

	sent := fakeNtfy(t, http.StatusOK)
	if err := Check(pageURL, "topic"); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(*sent) != 1 {
		t.Errorf("second run sent %d notifications, want 1", len(*sent))
	}
}
