package showtimes

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// sessionLinkPattern matches a bookable session link on a VOX movie page,
// e.g. href="https://uae.voxcinemas.com/booking/0039-145525". The page lists
// one such link per session, and has none until booking opens.
var sessionLinkPattern = regexp.MustCompile(`/booking/[0-9]+-[0-9]+`)

// hasShowtimesAvailable fetches the VOX movie page at pageURL and reports
// whether it lists at least one bookable session.
func hasShowtimesAvailable(pageURL string) (bool, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(pageURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%s returned status %d", pageURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	html := string(body)

	if sessionLinkPattern.MatchString(html) {
		return true, nil
	}

	// A client-rendered page never contains sessions in its HTML. Report it
	// as an error so a layout change does not look like "no showtimes".
	if strings.Contains(html, "/_next/static/") {
		return false, fmt.Errorf("%s is client-rendered; sessions are not in the HTML", pageURL)
	}

	return false, nil
}
