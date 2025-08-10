package main

import (
	"fmt"
	"os"

	"github.com/cr2007/vox-showtime-check/showtimes"
	_ "github.com/joho/godotenv/autoload"
)

// Monitors a webpage for showtime availability and sends notifications via ntfy.sh.
//
// It loads the last known state, checks the configured URL for showtime listings,
// sends notifications on changes or periodic "no showtimes" updates, and persists the updated state.
func main() {
	// Step 1: Read configuration from environment variables
	url := os.Getenv("SHOWTIMES_URL")
	topic := os.Getenv("NTFY_TOPIC")
	if url == "" || topic == "" {
		fmt.Println("❌ Missing SHOWTIMES_URL or NTFY_TOPIC env vars.")
		os.Exit(1)
	}

	showtimes.CheckShowtimeAvailability(url, topic)
}
