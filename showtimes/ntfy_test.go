package showtimes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ntfyRequest is one request that the fake ntfy.sh server received.
type ntfyRequest struct {
	Method, Path, Body string
	Header             http.Header
}

// fakeNtfy points ntfyBaseURL at a local server that replies with status
// and records each request. The original URL comes back after the test.
func fakeNtfy(t *testing.T, status int) *[]ntfyRequest {
	t.Helper()

	var got []ntfyRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, ntfyRequest{r.Method, r.URL.Path, string(body), r.Header})
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	original := ntfyBaseURL
	ntfyBaseURL = srv.URL
	t.Cleanup(func() { ntfyBaseURL = original })

	return &got
}

func TestNotificationSend(t *testing.T) {
	got := fakeNtfy(t, http.StatusOK)

	n := notification{Title: "Title text", Message: "Body text", Priority: "5"}
	if err := n.send("mytopic"); err != nil {
		t.Fatalf("send: %v", err)
	}

	if len(*got) != 1 {
		t.Fatalf("got %d requests, want 1", len(*got))
	}
	req := (*got)[0]
	if req.Method != http.MethodPost || req.Path != "/mytopic" || req.Body != "Body text" {
		t.Errorf("request = %s %s %q, want POST /mytopic %q", req.Method, req.Path, req.Body, "Body text")
	}
	if req.Header.Get("Title") != "Title text" || req.Header.Get("Priority") != "5" {
		t.Errorf("Title, Priority = %q, %q", req.Header.Get("Title"), req.Header.Get("Priority"))
	}
	if _, ok := req.Header["Tags"]; ok {
		t.Error("empty Tags field was sent as a header")
	}
}
