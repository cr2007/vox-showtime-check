package showtimes

import (
	"encoding/json"
	"os"
	"time"
)

// stateFile is the file that keeps the State between runs.
const stateFile = "state.json"

// Values of State.Status.
const (
	statusFound    = "found"
	statusNotFound = "not-found"
)

// State is the result of the last run.
type State struct {
	// Status is statusFound or statusNotFound.
	Status string `json:"status"`

	// LastNotFoundTs is the time of the last "not found" notification. The
	// zero value means "never", so the next "not found" check notifies.
	LastNotFoundTs time.Time `json:"last_not_found_ts,omitzero"`
}

// loadState reads the State from stateFile. A missing or invalid file gives
// a fresh "not found" State, so a lost cache causes at most one extra
// notification.
func loadState() State {
	var state State

	data, err := os.ReadFile(stateFile)
	if err != nil || json.Unmarshal(data, &state) != nil {
		return State{Status: statusNotFound}
	}

	return state
}

// saveState writes state to stateFile as indented JSON.
func saveState(state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0o644)
}
