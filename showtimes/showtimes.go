package showtimes

import (
	"fmt"
	"time"
)

// NotFoundInterval is how often a "still not found" notification is resent
// while no showtimes are available.
const NotFoundInterval = 2 * time.Hour

// notification is a single message to send via SendNotification.
type notification struct {
	Message string
	Headers map[string]string
}

// decideNotification is the pure decision logic behind
// CheckShowtimeAvailability: given the previous state, whether showtimes
// were found just now, and the current time, it returns the updated state
// and the notification to send (nil if none).
func decideNotification(state State, found bool, now time.Time, pageURL string) (State, *notification) {
	if found {
		var notify *notification
		if state.Status != "found" {
			notify = &notification{
				Message: fmt.Sprintf("🎬 Showtimes just appeared on %s", pageURL),
				Headers: map[string]string{
					"Title":    "Showtimes Available 🎉",
					"Priority": "5",
					"Tags":     "popcorn, clapper, vox-cinemas",
					"Actions":  fmt.Sprintf("view, Book now, %s", pageURL),
				},
			}
		}
		state.Status = "found"
		return state, notify
	}

	var notify *notification
	if now.Sub(state.LastNotFoundTs) >= NotFoundInterval {
		notify = &notification{
			Message: fmt.Sprintf("❌ Still no showtimes on %s", pageURL),
			Headers: map[string]string{
				"Title":    "No Showtimes yet",
				"Priority": "1",
			},
		}
		state.LastNotFoundTs = now
	}
	state.Status = "not-found"
	return state, notify
}

// CheckShowtimeAvailability checks pageURL for showtimes and notifies
// topic on ntfy.sh when availability changes. It returns an error if the
// page check fails; the state is then left unchanged.
func CheckShowtimeAvailability(pageURL, topic string) error {
	state := LoadState()

	found, err := hasShowtimesAvailable(pageURL)
	if err != nil {
		return fmt.Errorf("checking showtime availability: %w", err)
	}

	state, notify := decideNotification(state, found, time.Now(), pageURL)
	if notify != nil {
		SendNotification(topic, notify.Message, notify.Headers)
	}

	SaveState(state)
	return nil
}
