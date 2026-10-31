package showtimes

import (
	"fmt"
	"net/http"
	"strings"
)

// ntfyBaseURL is the ntfy.sh server. Tests point it at a local server.
var ntfyBaseURL = "https://ntfy.sh"

// notification is one ntfy.sh message. Each field except Message goes into
// the ntfy.sh header of the same name. Empty fields are not sent.
type notification struct {
	Title    string
	Message  string
	Priority string
	Tags     string
	Actions  string
}

// send posts n to the ntfy.sh topic. It returns an error if the request
// fails or ntfy.sh does not accept the message.
func (n notification) send(topic string) error {
	// Step 1: Make the request. The message is the body.
	req, err := http.NewRequest(http.MethodPost, ntfyBaseURL+"/"+topic, strings.NewReader(n.Message))
	if err != nil {
		return err
	}

	// Step 1.1: Add the ntfy.sh headers that have a value.
	for name, value := range map[string]string{
		"Title":    n.Title,
		"Priority": n.Priority,
		"Tags":     n.Tags,
		"Actions":  n.Actions,
	} {
		if value != "" {
			req.Header.Set(name, value)
		}
	}

	// Step 2: Send the request.
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Step 3: Make sure that ntfy.sh accepted the message.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy.sh returned status %d", resp.StatusCode)
	}

	return nil
}
