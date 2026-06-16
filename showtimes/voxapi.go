package showtimes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// chunkURLPattern matches references to VOX's Next.js static chunks in a
// page's HTML.
var chunkURLPattern = regexp.MustCompile(`/_next/static/chunks/[A-Za-z0-9._-]+\.js`)

// apiKeyPattern matches VOX's frontend API key as embedded in a JS chunk,
// e.g. apiKey:"AbC123...". It's public (every visitor's browser gets the
// same value), but VOX changes the chunk's hash on every deploy, so this
// reads the key fresh instead of hardcoding it.
var apiKeyPattern = regexp.MustCompile(`apiKey:"([A-Za-z0-9_-]+)"`)

type movieInfoResponse struct {
	Bookable bool   `json:"Bookable"`
	HoCode   string `json:"HoCode"`
}

type sessionsResponse struct {
	Cinemas []struct {
		SessionGroups []struct {
			Sessions []struct {
				Showtime string `json:"showtime"`
			} `json:"sessions"`
		} `json:"sessionGroups"`
	} `json:"cinemas"`
}

// hasSessions reports whether any cinema has at least one session.
func (s sessionsResponse) hasSessions() bool {
	for _, cinema := range s.Cinemas {
		for _, group := range cinema.SessionGroups {
			if len(group.Sessions) > 0 {
				return true
			}
		}
	}
	return false
}

// parseMovieURL extracts the subdomain (e.g. "uae") and movie slug (e.g.
// "obsession") from a VOX movie page URL such as
// https://uae.voxcinemas.com/movies/obsession.
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

// extractAPIKeyFromChunk fetches a single JS chunk (resolved against
// pageURL's scheme and host) and pulls the API key out of it, if present.
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
		return "", fmt.Errorf("api key pattern not found in %s", chunkURL)
	}

	return m[1], nil
}

// discoverAPIKey finds VOX's frontend API key by scanning the movie page's
// own JS chunks. state.CachedAPIKeyChunk is tried first as a shortcut; a
// miss (e.g. after a VOX deploy) triggers a full rescan, and the chunk
// that works is cached for next time.
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

	return "", fmt.Errorf("could not find api key in any chunk referenced by %s", pageURL)
}

// fetchAuthToken obtains a short-lived guest bearer token. No credentials
// are required; this is what any anonymous browser receives.
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

// hasShowtimesAvailable checks whether the movie at pageURL has any
// sessions today, by calling VOX's own API in the same way its frontend
// does: discover the API key, get a guest token, resolve the movie's
// HoCode, then fetch today's sessions.
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

	return sessions.hasSessions(), nil
}
