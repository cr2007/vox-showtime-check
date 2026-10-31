// Command vox-showtime-check checks a VOX Cinemas movie page one time and
// sends an ntfy.sh notification when showtimes appear.
//
// It reads SHOWTIMES_URL and NTFY_TOPIC from the environment or from a .env
// file. It exits with code 1 if a value is missing or the check fails.
package main

import (
	"fmt"
	"os"

	"github.com/cr2007/vox-showtime-check/showtimes"
	_ "github.com/joho/godotenv/autoload"
)

// main reads the configuration and runs one check.
func main() {
	// Step 1: Read the configuration.
	pageURL := os.Getenv("SHOWTIMES_URL")
	topic := os.Getenv("NTFY_TOPIC")
	if pageURL == "" || topic == "" {
		fmt.Fprintln(os.Stderr, "error: set SHOWTIMES_URL and NTFY_TOPIC")
		os.Exit(1)
	}

	// Step 2: Run one check.
	if err := showtimes.Check(pageURL, topic); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
