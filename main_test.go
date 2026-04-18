package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeHandlerRendersVersionAndAuthors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"type":"single","joke":"Test joke"}`)
	}))
	defer server.Close()

	oldClient := httpClient
	oldJokeAPIURL := jokeAPIURL
	httpClient = server.Client()
	jokeAPIURL = server.URL
	t.Cleanup(func() {
		httpClient = oldClient
		jokeAPIURL = oldJokeAPIURL
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	homeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	for _, expected := range []string{
		"Application Version: " + version,
		"Muhammad Safwan Karim",
		"Muhammad Jahanzaib",
		"Muhamaad Mueed",
		"Waseem Gul",
		"Baig",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected response to contain %q", expected)
		}
	}
}

func TestFetchJokeFallbackOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `invalid-json`)
	}))
	defer server.Close()

	oldClient := httpClient
	oldJokeAPIURL := jokeAPIURL
	httpClient = server.Client()
	jokeAPIURL = server.URL
	t.Cleanup(func() {
		httpClient = oldClient
		jokeAPIURL = oldJokeAPIURL
	})

	joke := fetchJoke(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	if joke != fallbackJoke {
		t.Fatalf("expected fallback joke, got %q", joke)
	}
}
