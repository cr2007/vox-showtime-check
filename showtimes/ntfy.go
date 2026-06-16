package showtimes

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ntfyBaseURL is overridden in tests to point at a mock server.
var ntfyBaseURL = "https://ntfy.sh"

// SendNotification posts msg to a ntfy.sh topic with optional headers
// (e.g. Title, Priority, Tags). It's a no-op if topic is empty.
func SendNotification(topic, msg string, headers map[string]string) {
	if topic == "" {
		fmt.Println("NTFY_TOPIC not configured. Skipping notification.")
		return
	}

	req, err := http.NewRequest("POST", ntfyBaseURL+"/"+topic, strings.NewReader(msg))
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
