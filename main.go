package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type State struct {
	Status         string    `json:"status"`            // "found" or "not found"
	LastNotFoundTs time.Time `json:"last_not_found_ts"` // Timestamp for last "not found" message
}

const stateFile = "state.json"
const notFoundInterval = 2 * time.Hour

func loadState() State {
	var state State
	data, err := os.ReadFile(stateFile)
	if err != nil {
		state.Status = "not-found"
		state.LastNotFoundTs = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		return state
	}

	json.Unmarshal(data, &state)
	return state
}

func saveState(state State) {
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)
}

func sendNotification(topic, msg string, headers map[string]string) {
	if topic == "" {
		fmt.Println("NTFY_TOPIC not configured. Skipping notification.")
		return
	}

	url := "https://ntfy.sh/" + topic
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

func main() {
	url := os.Getenv("SHOWTIMES_URL")
	topic := os.Getenv("NTFY_TOPIC")
	if url == "" || topic == "" {
		fmt.Println("❌ Missing SHOWTIMES_URL or NTFY_TOPIC env vars.")
		os.Exit(1)
	}

	state := loadState()
	now := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching page:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	found := strings.Contains(body, `id="showtimes"`)

	if found && state.Status != "found" {
		sendNotification(topic,
			fmt.Sprintf("🎬 Showtimes just appeared on %s", url),
			map[string]string{
				"Title":    "Showtimes Available 🎉",
				"Priority": "5",
				"Tags":     "popcorn, clapper, vox-cinemas",
				"Actions":  fmt.Sprintf("view, Book now, %s", url),
			})
		state.Status = "found"
	} else if !found {
		if now.Sub(state.LastNotFoundTs) >= notFoundInterval {
			sendNotification(topic,
				fmt.Sprintf("❌ Still no showtimes on %s", url),
				map[string]string{
					"Title":    "No Showtimes yet",
					"Priority": "1",
				})
			state.LastNotFoundTs = now
		}
		state.Status = "not-found"
	}

	saveState(state)
}
