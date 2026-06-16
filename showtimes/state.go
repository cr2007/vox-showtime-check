package showtimes

import (
	"encoding/json"
	"os"
	"time"
)

// State tracks whether showtimes were last seen, when the last "not found"
// notification was sent, and which JS chunk last yielded the VOX API key.
type State struct {
	Status            string    `json:"status"` // "found" or "not-found"
	LastNotFoundTs    time.Time `json:"last_not_found_ts"`
	CachedAPIKeyChunk string    `json:"cached_api_key_chunk,omitempty"`
}

const stateFile = "state.json"

// epoch is a "long ago" default so a fresh state is always overdue for a
// "not found" notification.
var epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

// LoadState reads state.json, or returns a fresh "not-found" state if it
// doesn't exist or can't be parsed.
func LoadState() State {
	state := State{Status: "not-found", LastNotFoundTs: epoch}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return state
	}

	json.Unmarshal(data, &state)

	// A state.json missing this field (e.g. hand-edited, or from an older
	// version) would otherwise leave it at Go's zero time.
	if state.LastNotFoundTs.IsZero() {
		state.LastNotFoundTs = epoch
	}

	return state
}

// SaveState writes state as formatted JSON to state.json.
func SaveState(state State) {
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)
}
