package showtimes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseMovieURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantSubdomain string
		wantSlug      string
		wantErr       bool
	}{
		{
			name:          "typical movie url",
			url:           "https://uae.voxcinemas.com/movies/backrooms",
			wantSubdomain: "uae",
			wantSlug:      "backrooms",
		},
		{
			name:          "trailing slash",
			url:           "https://ksa.voxcinemas.com/movies/backrooms/",
			wantSubdomain: "ksa",
			wantSlug:      "backrooms",
		},
		{
			name:    "no path",
			url:     "https://uae.voxcinemas.com/",
			wantErr: true,
		},
		{
			name:    "invalid url",
			url:     "http://[::1]:namedport",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subdomain, slug, err := parseMovieURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if subdomain != tt.wantSubdomain {
				t.Errorf("subdomain = %q, want %q", subdomain, tt.wantSubdomain)
			}
			if slug != tt.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tt.wantSlug)
			}
		})
	}
}

func TestSessionsResponseHasSessions(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{name: "no cinemas", json: `{"cinemas":[]}`, want: false},
		{
			name: "cinema with empty session groups",
			json: `{"cinemas":[{"sessionGroups":[]}]}`,
			want: false,
		},
		{
			name: "session group with no sessions",
			json: `{"cinemas":[{"sessionGroups":[{"sessions":[]}]}]}`,
			want: false,
		},
		{
			name: "one session present",
			json: `{"cinemas":[{"sessionGroups":[{"sessions":[{"showtime":"2026-06-16T14:00:00+00:00"}]}]}]}`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sessions sessionsResponse
			if err := json.Unmarshal([]byte(tt.json), &sessions); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if got := sessions.hasSessions(); got != tt.want {
				t.Errorf("hasSessions() = %v, want %v", got, tt.want)
			}
		})
	}
}

// newChunkTestServer serves pageHTML at "/" and each chunk's content at its
// given path, mimicking a VOX movie page referencing JS chunks.
func newChunkTestServer(t *testing.T, pageHTML string, chunks map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pageHTML))
	})
	for path, content := range chunks {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(content))
		})
	}
	return httptest.NewServer(mux)
}

func TestDiscoverAPIKeyFullScan(t *testing.T) {
	pageHTML := `<script src="/_next/static/chunks/123-abc.js"></script>`
	chunks := map[string]string{
		"/_next/static/chunks/123-abc.js": `var x=1;apiKey:"secret-key-1";`,
	}
	server := newChunkTestServer(t, pageHTML, chunks)
	defer server.Close()

	state := &State{}
	key, err := discoverAPIKey(server.Client(), server.URL+"/", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "secret-key-1" {
		t.Errorf("key = %q, want %q", key, "secret-key-1")
	}
	if state.CachedAPIKeyChunk != "/_next/static/chunks/123-abc.js" {
		t.Errorf("CachedAPIKeyChunk = %q, want the discovered chunk", state.CachedAPIKeyChunk)
	}
}

func TestDiscoverAPIKeyUsesCacheWithoutRescanning(t *testing.T) {
	pageHitCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pageHitCount++
		w.Write([]byte(`<script src="/_next/static/chunks/should-not-be-used.js"></script>`))
	})
	mux.HandleFunc("/_next/static/chunks/cached.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`apiKey:"cached-key";`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	state := &State{CachedAPIKeyChunk: "/_next/static/chunks/cached.js"}
	key, err := discoverAPIKey(server.Client(), server.URL+"/", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "cached-key" {
		t.Errorf("key = %q, want %q", key, "cached-key")
	}
	if pageHitCount != 0 {
		t.Errorf("page was fetched %d times, want 0 (cache should avoid a rescan)", pageHitCount)
	}
}

func TestDiscoverAPIKeyRescansOnStaleCacheMiss(t *testing.T) {
	pageHTML := `<script src="/_next/static/chunks/new.js"></script>`
	chunks := map[string]string{
		"/_next/static/chunks/new.js": `apiKey:"new-key";`,
	}
	server := newChunkTestServer(t, pageHTML, chunks)
	defer server.Close()

	state := &State{CachedAPIKeyChunk: "/_next/static/chunks/gone-after-deploy.js"}
	key, err := discoverAPIKey(server.Client(), server.URL+"/", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "new-key" {
		t.Errorf("key = %q, want %q", key, "new-key")
	}
	if state.CachedAPIKeyChunk != "/_next/static/chunks/new.js" {
		t.Errorf("CachedAPIKeyChunk = %q, want it updated to the new chunk", state.CachedAPIKeyChunk)
	}
}

func TestFetchAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"access_token": "tok123"})
	}))
	defer server.Close()

	token, err := fetchAuthToken(server.Client(), server.URL, "https://uae.voxcinemas.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "tok123" {
		t.Errorf("token = %q, want %q", token, "tok123")
	}
}

func TestFetchAuthTokenMissingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	_, err := fetchAuthToken(server.Client(), server.URL, "https://uae.voxcinemas.com/")
	if err == nil {
		t.Fatal("expected an error for a missing access_token")
	}
}

func TestApiGetSetsExpectedHeaders(t *testing.T) {
	var gotAuth, gotKey, gotAuthType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("x-api-key")
		gotAuthType = r.Header.Get("x-auth-type")
		json.NewEncoder(w).Encode(map[string]bool{"Bookable": true})
	}))
	defer server.Close()

	var out movieInfoResponse
	err := apiGet(server.Client(), server.URL, "tok", "key", "https://uae.voxcinemas.com/", &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
	if gotKey != "key" {
		t.Errorf("x-api-key = %q, want %q", gotKey, "key")
	}
	if gotAuthType != "Oauth" {
		t.Errorf("x-auth-type = %q, want %q", gotAuthType, "Oauth")
	}
}

func TestApiGetNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var out movieInfoResponse
	err := apiGet(server.Client(), server.URL, "tok", "key", "https://uae.voxcinemas.com/", &out)
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}
