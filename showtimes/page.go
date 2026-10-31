package showtimes

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// sessionLinkPattern matches a session link on a VOX movie page, for example
// "/booking/0039-145525". The page has one link for each session, and no
// links before booking opens.
var sessionLinkPattern = regexp.MustCompile(`/booking/[0-9]+-[0-9]+`)

// hasSessions gets the VOX movie page at pageURL and tells if it has at
// least one session link.
func hasSessions(pageURL string) (bool, error) {
	// Step 1: Get the page.
	resp, err := httpClient.Get(pageURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Step 1.1: An error status (for example 404 for a wrong slug) is not
	// the same as "no showtimes".
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	html := string(body)

	// Step 2: Look for a session link.
	if sessionLinkPattern.MatchString(html) {
		return true, nil
	}

	// Step 2.1: A client-rendered page never has sessions in its HTML. Return
	// an error, so that a layout change does not look like "no showtimes".
	if strings.Contains(html, "/_next/static/") {
		return false, errors.New("page is client-rendered, so its HTML has no sessions")
	}

	return false, nil
}
