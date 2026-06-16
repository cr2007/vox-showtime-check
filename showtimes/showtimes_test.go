package showtimes

import (
	"testing"
	"time"
)

func TestDecideNotification(t *testing.T) {
	const pageURL = "https://uae.voxcinemas.com/movies/backrooms"
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		state      State
		found      bool
		wantNotify bool
		wantStatus string
		wantTs     time.Time
	}{
		{
			name:       "transition to found notifies",
			state:      State{Status: "not-found", LastNotFoundTs: epoch},
			found:      true,
			wantNotify: true,
			wantStatus: "found",
			wantTs:     epoch,
		},
		{
			name:       "still found does not re-notify",
			state:      State{Status: "found", LastNotFoundTs: epoch},
			found:      true,
			wantNotify: false,
			wantStatus: "found",
			wantTs:     epoch,
		},
		{
			name:       "not found, interval elapsed notifies",
			state:      State{Status: "found", LastNotFoundTs: now.Add(-NotFoundInterval)},
			found:      false,
			wantNotify: true,
			wantStatus: "not-found",
			wantTs:     now,
		},
		{
			name:       "not found, interval not yet elapsed stays quiet",
			state:      State{Status: "not-found", LastNotFoundTs: now.Add(-time.Minute)},
			found:      false,
			wantNotify: false,
			wantStatus: "not-found",
			wantTs:     now.Add(-time.Minute),
		},
		{
			name:       "first ever not-found check notifies immediately",
			state:      State{Status: "not-found", LastNotFoundTs: epoch},
			found:      false,
			wantNotify: true,
			wantStatus: "not-found",
			wantTs:     now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newState, notify := decideNotification(tt.state, tt.found, now, pageURL)

			if (notify != nil) != tt.wantNotify {
				t.Errorf("notify = %v, want notify present = %v", notify, tt.wantNotify)
			}
			if newState.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", newState.Status, tt.wantStatus)
			}
			if !newState.LastNotFoundTs.Equal(tt.wantTs) {
				t.Errorf("LastNotFoundTs = %v, want %v", newState.LastNotFoundTs, tt.wantTs)
			}
		})
	}
}

func TestDecideNotificationMessageContent(t *testing.T) {
	now := time.Now()
	pageURL := "https://uae.voxcinemas.com/movies/backrooms"

	_, notify := decideNotification(State{Status: "not-found", LastNotFoundTs: epoch}, true, now, pageURL)
	if notify == nil {
		t.Fatal("expected a notification")
	}
	if notify.Headers["Title"] != "Showtimes Available 🎉" {
		t.Errorf("Title = %q", notify.Headers["Title"])
	}

	_, notify = decideNotification(State{Status: "found", LastNotFoundTs: epoch}, false, now, pageURL)
	if notify == nil {
		t.Fatal("expected a notification")
	}
	if notify.Headers["Title"] != "No Showtimes yet" {
		t.Errorf("Title = %q", notify.Headers["Title"])
	}
}
