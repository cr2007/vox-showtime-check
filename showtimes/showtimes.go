package showtimes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// State represents the current status and the timestamp of the last "not found" event.
type State struct {
	Status         string    `json:"status"`            // "found" or "not found"
	LastNotFoundTs time.Time `json:"last_not_found_ts"` // Timestamp for last "not found" message
}

const stateFile = "state.json"
const NotFoundInterval = 2 * time.Hour

// Retrieves the saved State from disk, or returns a default if unavailable.
func LoadState() State {
	var state State

	// Step 1: Read the state file from disk.
	data, err := os.ReadFile(stateFile)
	if err != nil {
		// Step 2: If reading fails, return a default "not-found" state with an epoch timestamp.
		state.Status = "not-found"
		state.LastNotFoundTs = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		return state
	}

	// Step 3: Parse JSON data into the State struct.
	json.Unmarshal(data, &state)

	// Step 4: Return the loaded state.
	return state
}

// Stores the given State as formatted JSON in a file.
func SaveState(state State) {
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)
}

// Posts a message to an ntfy.sh topic with optional headers.
//
// Parameters:
//   - topic:   The ntfy.sh topic name. Must be non-empty to send a notification.
//   - msg:     The notification message body.
//   - headers: Optional key-value pairs for HTTP headers to include in the request.
//
// Example usage:
//   sendNotification(
//       "alerts",
//       "Disk space is low",
//       map[string]string{"Title": "Server Warning", "Priority": "high"},
//   )
func SendNotification(topic, msg string, headers map[string]string) {
	if topic == "" {
		fmt.Println("NTFY_TOPIC not configured. Skipping notification.")
		return
	}

	url := fmt.Sprintf("https://ntfy.sh/%s", topic)
	req, err := http.NewRequest("POST", url, strings.NewReader(msg))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending notification:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("ntfy.sh returned status %d\n", resp.StatusCode)
	}
}

// CheckShowtimeAvailability checks a URL for showtime listings and sends notifications.
func CheckShowtimeAvailability(url, topic string) {
	// Step 1: Load the previous state from disk
	state := LoadState()
	now := time.Now()

	// Step 2: Fetch the target webpage
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching page:", err)
		return
	}
	defer resp.Body.Close()

	// Step 4: Read and check page content for showtime availability
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	found := strings.Contains(body, `id="showtimes"`)

	// Step 5: If showtimes are found and status changed, send a "showtimes available" notification.
	if found && state.Status != "found" {
		SendNotification(topic,
			fmt.Sprintf("🎬 Showtimes just appeared on %s", url),
			map[string]string{
				"Title":    "Showtimes Available 🎉",
				"Priority": "5",
				"Tags":     "popcorn, clapper, vox-cinemas",
				"Actions":  fmt.Sprintf("view, Book now, %s", url),
			})
		state.Status = "found"

	// Step 6: If no showtimes, optionally send a periodic "no showtimes" notification.
	} else if !found {
		if now.Sub(state.LastNotFoundTs) >= NotFoundInterval {
			SendNotification(topic,
				fmt.Sprintf("❌ Still no showtimes on %s", url),
				map[string]string{
					"Title":    "No Showtimes yet",
					"Priority": "1",
				})
			state.LastNotFoundTs = now
		}
		state.Status = "not-found"
	}

	// Step 7: Save the updated state to disk.
	SaveState(state)
}
