package showtimes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// showingPage is a trimmed copy of a real VOX movie page with sessions.
const showingPage = `<html><body>
<section id="showtimes">
  <h3 class="highlight">Al Jimi Mall</h3>
  <ol class="showtimes"><li><strong>MAX</strong><ol>
    <li data-id="0039-145525">
      <a class="action showtime" href="https://uae.voxcinemas.com/booking/0039-145525" data-id="0039-145525">10:45am </a>
    </li>
  </ol></li></ol>
</section>
</body></html>`

// comingSoonPage is a movie page with no sessions yet.
const comingSoonPage = `<html><body>
<h1>Children of Blood and Bone</h1>
<a href="/movies/comingsoon">Coming Soon</a>
</body></html>`

// nextJSShell is the client-rendered layout that VOX used before.
const nextJSShell = `<html><head>
<script src="/_next/static/chunks/main-abc123.js"></script>
</head><body><div id="__next"></div></body></html>`

// servePage starts a local server that replies to each request with status
// and body. It returns the URL of a movie page on that server.
func servePage(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/movies/some-movie"
}

func TestHasSessions(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		want      bool
		wantErrIn string
	}{
		{name: "page with sessions", status: http.StatusOK, body: showingPage, want: true},
		{name: "coming soon page", status: http.StatusOK, body: comingSoonPage, want: false},
		{name: "unknown movie", status: http.StatusNotFound, body: "not found", wantErrIn: "status 404"},
		{name: "client-rendered layout", status: http.StatusOK, body: nextJSShell, wantErrIn: "client-rendered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hasSessions(servePage(t, tt.status, tt.body))

			if tt.wantErrIn != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrIn) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErrIn)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
