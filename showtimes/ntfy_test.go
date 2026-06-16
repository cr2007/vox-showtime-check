package showtimes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendNotificationPostsExpectedRequest(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	var gotHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHeaders = r.Header
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restore := setNtfyBaseURL(server.URL)
	defer restore()

	SendNotification("mytopic", "hello world", map[string]string{"Title": "Test"})

	if gotMethod != http.MethodPost {
		t.Errorf("Method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/mytopic" {
		t.Errorf("Path = %q, want %q", gotPath, "/mytopic")
	}
	if gotBody != "hello world" {
		t.Errorf("Body = %q, want %q", gotBody, "hello world")
	}
	if got := gotHeaders.Get("Title"); got != "Test" {
		t.Errorf("Title header = %q, want %q", got, "Test")
	}
}

func TestSendNotificationSkipsWhenTopicEmpty(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	restore := setNtfyBaseURL(server.URL)
	defer restore()

	SendNotification("", "hello world", nil)

	if called {
		t.Error("expected no request to be made when topic is empty")
	}
}

// setNtfyBaseURL overrides ntfyBaseURL for the duration of a test and
// returns a func to restore it.
func setNtfyBaseURL(url string) func() {
	original := ntfyBaseURL
	ntfyBaseURL = url
	return func() { ntfyBaseURL = original }
}
