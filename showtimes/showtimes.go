package showtimes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// State represents the current status and the timestamp of the last "not found" event.
type State struct {
	Status            string    `json:"status"`                       // "found" or "not found"
	LastNotFoundTs    time.Time `json:"last_not_found_ts"`             // Timestamp for last "not found" message
	CachedAPIKeyChunk string    `json:"cached_api_key_chunk,omitempty"` // Last JS chunk path the x-api-key was found in
}

const stateFile = "state.json"
const NotFoundInterval = 2 * time.Hour

// chunkURLPattern matches script references to VOX's Next.js static chunks
// in a page's HTML.
var chunkURLPattern = regexp.MustCompile(`/_next/static/chunks/[A-Za-z0-9._-]+\.js`)

// apiKeyPattern matches the literal "x-api-key" VOX's frontend embeds in one
// of its JS chunks, e.g. `apiKey:"AbC123..."`. It's the same value every
// visitor's browser receives — not a per-session secret — but VOX rotates
// the chunk's content hash (and potentially the key itself) on every
// deploy, so rather than hardcoding the key we rediscover it the same way a
// browser would: by reading it out of the page's own JS.
var apiKeyPattern = regexp.MustCompile(`apiKey:"([A-Za-z0-9_-]+)"`)

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

	// Step 4: Normalize a missing/zero-value timestamp (e.g. an older or
	// hand-edited state.json without this field) to the epoch, so it
	// reads sensibly and still compares as "long overdue" against
	// NotFoundInterval.
	if state.LastNotFoundTs.IsZero() {
		state.LastNotFoundTs = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	// Step 5: Return the loaded state.
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

	ntfyURL := fmt.Sprintf("https://ntfy.sh/%s", topic)
	req, err := http.NewRequest("POST", ntfyURL, strings.NewReader(msg))
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

// movieInfoResponse mirrors the relevant fields of VOX's
// /v1/vox2-0/groups/Movies/{slug} response.
type movieInfoResponse struct {
	Bookable bool   `json:"Bookable"`
	HoCode   string `json:"HoCode"`
}

// sessionsResponse mirrors the relevant fields of VOX's
// /v1/vox2-0/groups/api/Sessions/{region}/{hoCode}/{date} response.
type sessionsResponse struct {
	Cinemas []struct {
		SessionGroups []struct {
			Sessions []struct {
				Showtime string `json:"showtime"`
			} `json:"sessions"`
		} `json:"sessionGroups"`
	} `json:"cinemas"`
}

// parseMovieURL extracts the regional subdomain (e.g. "uae") and the movie
// slug (e.g. "obsession") from a VOX Cinemas movie page URL such as
// https://uae.voxcinemas.com/movies/<movie>.
func parseMovieURL(pageURL string) (subdomain, slug string, err error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", "", err
	}

	subdomain = strings.SplitN(u.Hostname(), ".", 2)[0]
	if subdomain == "" {
		return "", "", fmt.Errorf("could not determine subdomain from host %q", u.Hostname())
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	slug = segments[len(segments)-1]
	if slug == "" {
		return "", "", fmt.Errorf("could not determine movie slug from path %q", u.Path)
	}

	return subdomain, slug, nil
}

// extractAPIKeyFromChunk fetches a single JS chunk (relative to pageURL's
// host) and tries to pull the x-api-key out of it.
func extractAPIKeyFromChunk(client *http.Client, pageURL, chunkPath string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	chunkURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, chunkPath)

	resp, err := client.Get(chunkURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chunk %s returned status %d", chunkURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	m := apiKeyPattern.FindStringSubmatch(string(body))
	if m == nil {
		return "", fmt.Errorf("apiKey pattern not found in %s", chunkURL)
	}

	return m[1], nil
}

// discoverAPIKey finds VOX's frontend "x-api-key" by inspecting the movie
// page's own JavaScript bundles — the same way a browser obtains it,
// instead of hardcoding a value that VOX can rotate at any time.
//
// To avoid re-fetching the page and scanning every chunk on every run, the
// chunk that last yielded the key is cached on state.CachedAPIKeyChunk and
// tried first. A cache miss (e.g. after a VOX deploy changes the chunk's
// content hash) triggers a full rescan, and the new chunk is cached for
// next time.
func discoverAPIKey(client *http.Client, pageURL string, state *State) (string, error) {
	if state.CachedAPIKeyChunk != "" {
		if key, err := extractAPIKeyFromChunk(client, pageURL, state.CachedAPIKeyChunk); err == nil {
			return key, nil
		}
		state.CachedAPIKeyChunk = ""
	}

	resp, err := client.Get(pageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	for _, chunk := range chunkURLPattern.FindAllString(string(body), -1) {
		if key, err := extractAPIKeyFromChunk(client, pageURL, chunk); err == nil {
			state.CachedAPIKeyChunk = chunk
			return key, nil
		}
	}

	return "", fmt.Errorf("could not find x-api-key in any chunk referenced by %s", pageURL)
}

// fetchAuthToken obtains a short-lived guest bearer token from VOX's API.
// No credentials are required — this mirrors what any anonymous visitor's
// browser receives.
func fetchAuthToken(client *http.Client, apiBase, referer string) (string, error) {
	req, err := http.NewRequest("GET", apiBase+"/groups/authToken", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("Accept-Language", "en")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authToken request returned status %d", resp.StatusCode)
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("authToken response missing access_token")
	}

	return out.AccessToken, nil
}

// apiGet performs an authenticated GET against VOX's API and decodes the
// JSON response into out.
func apiGet(client *http.Client, apiURL, token, apiKey, referer string, out any) error {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("x-auth-type", "Oauth")
	req.Header.Set("Referer", referer)
	req.Header.Set("Accept-Language", "en")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request to %s returned status %d", apiURL, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// hasShowtimesAvailable queries VOX's internal API directly to determine
// whether the movie at pageURL has any bookable sessions today.
//
// The VOX site is a client-side rendered (Next.js) app: the showtimes block
// is populated by JavaScript after the page loads, so a plain HTML fetch
// never contains it. This calls the same backend API the site's own
// frontend calls, in the same steps:
//  1. Discover the frontend's x-api-key from the page's own JS (see discoverAPIKey).
//  2. Get a guest auth token.
//  3. Resolve the movie slug to its internal "HoCode".
//  4. Fetch today's sessions for that HoCode and check if any exist.
//
// state is passed in (rather than just read) because step 1 may update its
// API-key chunk cache as a side effect.
func hasShowtimesAvailable(pageURL string, state *State) (bool, error) {
	subdomain, slug, err := parseMovieURL(pageURL)
	if err != nil {
		return false, err
	}

	region := strings.ToUpper(subdomain)
	apiBase := fmt.Sprintf("https://%s-apife.voxcinemas.com", subdomain)
	referer := fmt.Sprintf("https://%s.voxcinemas.com/", subdomain)

	client := &http.Client{Timeout: 15 * time.Second}

	apiKey, err := discoverAPIKey(client, pageURL, state)
	if err != nil {
		return false, fmt.Errorf("discovering api key: %w", err)
	}

	token, err := fetchAuthToken(client, apiBase, referer)
	if err != nil {
		return false, fmt.Errorf("fetching auth token: %w", err)
	}

	var movie movieInfoResponse
	movieURL := fmt.Sprintf("%s/v1/vox2-0/groups/Movies/%s", apiBase, slug)
	if err := apiGet(client, movieURL, token, apiKey, referer, &movie); err != nil {
		return false, fmt.Errorf("fetching movie info: %w", err)
	}
	if !movie.Bookable || movie.HoCode == "" {
		return false, nil
	}

	today := time.Now().UTC().Format("2006-01-02")
	sessionsURL := fmt.Sprintf("%s/v1/vox2-0/groups/api/Sessions/%s/%s/%s", apiBase, region, movie.HoCode, today)
	var sessions sessionsResponse
	if err := apiGet(client, sessionsURL, token, apiKey, referer, &sessions); err != nil {
		return false, fmt.Errorf("fetching sessions: %w", err)
	}

	for _, cinema := range sessions.Cinemas {
		for _, group := range cinema.SessionGroups {
			if len(group.Sessions) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

// CheckShowtimeAvailability checks a URL for showtime listings and sends notifications.
func CheckShowtimeAvailability(pageURL, topic string) {
	// Step 1: Load the previous state from disk
	state := LoadState()
	now := time.Now()

	// Step 2: Query VOX's API for today's sessions
	found, err := hasShowtimesAvailable(pageURL, &state)
	if err != nil {
		fmt.Println("Error checking showtime availability:", err)
		return
	}

	// Step 3: If showtimes are found and status changed, send a "showtimes available" notification.
	if found && state.Status != "found" {
		SendNotification(topic,
			fmt.Sprintf("🎬 Showtimes just appeared on %s", pageURL),
			map[string]string{
				"Title":    "Showtimes Available 🎉",
				"Priority": "5",
				"Tags":     "popcorn, clapper, vox-cinemas",
				"Actions":  fmt.Sprintf("view, Book now, %s", pageURL),
			})
		state.Status = "found"

	// Step 4: If no showtimes, optionally send a periodic "no showtimes" notification.
	} else if !found {
		if now.Sub(state.LastNotFoundTs) >= NotFoundInterval {
			SendNotification(topic,
				fmt.Sprintf("❌ Still no showtimes on %s", pageURL),
				map[string]string{
					"Title":    "No Showtimes yet",
					"Priority": "1",
				})
			state.LastNotFoundTs = now
		}
		state.Status = "not-found"
	}

	// Step 5: Save the updated state to disk.
	SaveState(state)
}
