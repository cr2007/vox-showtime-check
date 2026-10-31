// Package showtimes checks a VOX Cinemas movie page for showtimes and sends
// ntfy.sh notifications when the result changes.
package showtimes

import (
	"fmt"
	"net/http"
	"time"
)

// NotFoundInterval is the minimum time between two "still no showtimes"
// notifications.
const NotFoundInterval = 2 * time.Hour

// httpClient is the client for the VOX page and for ntfy.sh.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// Check runs one check of the VOX movie page at pageURL and sends a
// notification to the ntfy.sh topic when necessary. If the check or the
// notification fails, Check returns an error and keeps the old state, so
// the next run tries again.
func Check(pageURL, topic string) error {
	// Step 1: Load the state of the last run.
	state := loadState()

	// Step 2: Check the page for sessions.
	found, err := hasSessions(pageURL)
	if err != nil {
		return fmt.Errorf("checking %s: %w", pageURL, err)
	}

	// Step 3: Get the new state and the notification to send.
	state, n := decide(state, found, time.Now(), pageURL)

	// Step 4: Send the notification, if there is one.
	if n != nil {
		if err := n.send(topic); err != nil {
			return fmt.Errorf("sending notification: %w", err)
		}
	}

	// Step 5: Save the new state for the next run.
	return saveState(state)
}

// decide returns the new state after a check, and the notification to send.
// The notification is nil when there is nothing new to tell.
func decide(state State, found bool, now time.Time, pageURL string) (State, *notification) {
	// Step 1: Showtimes are available.
	if found {
		// Step 1.1: Notify only when the status changes to "found".
		var n *notification
		if state.Status != statusFound {
			n = &notification{
				Title:    "Showtimes available",
				Message:  "Showtimes just appeared on " + pageURL,
				Priority: "5",
				Tags:     "popcorn,clapper,vox-cinemas",
				Actions:  "view, Book now, " + pageURL,
			}
		}
		state.Status = statusFound
		return state, n
	}

	// Step 2: No showtimes.
	state.Status = statusNotFound

	// Step 2.1: Notify at most once in each NotFoundInterval.
	if now.Sub(state.LastNotFoundTs) < NotFoundInterval {
		return state, nil
	}
	state.LastNotFoundTs = now

	return state, &notification{
		Title:    "No showtimes yet",
		Message:  "Still no showtimes on " + pageURL,
		Priority: "1",
	}
}
