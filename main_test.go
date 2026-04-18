package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// seedCache pre-populates the global joke cache for tests so no real HTTP
// calls are made to the JokeAPI.
func seedCache(jokes ...string) {
	jokeCache.mu.Lock()
	jokeCache.items = append([]string(nil), jokes...)
	jokeCache.refilling = false
	jokeCache.mu.Unlock()
}

func TestHomeHandlerRendersVersionAndAuthors(t *testing.T) {
	// Seed enough jokes so the handler serving one doesn't trigger a background refill.
	seedCache("Test joke 1", "Test joke 2", "Test joke 3")

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

func TestJokeCacheNextDrainsAndRefills(t *testing.T) {
	// Set up a mock API server that returns two jokes.
	apiCalled := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiCalled++
		w.Header().Set("Content-Type", "application/json")
		resp := batchJokeResponse{Jokes: []jokeResponse{
			{Type: "single", Joke: "Refill joke 1"},
			{Type: "single", Joke: "Refill joke 2"},
		}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldClient := httpClient
	oldURL := jokeAPIURL
	httpClient = server.Client()
	jokeAPIURL = server.URL
	t.Cleanup(func() {
		httpClient = oldClient
		jokeAPIURL = oldURL
		// Reset cache so subsequent tests start clean.
		seedCache()
	})

	// Seed exactly one joke so the first next() returns it and triggers refill.
	seedCache("Seeded joke")

	first := jokeCache.next()
	if first != "Seeded joke" {
		t.Fatalf("expected seeded joke, got %q", first)
	}

	// Give the background goroutine time to complete (it calls a local httptest server).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		jokeCache.mu.Lock()
		n := len(jokeCache.items)
		jokeCache.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if apiCalled == 0 {
		t.Fatal("expected background refill to have called the API")
	}
}

func TestFetchBatchJokesFallbackOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `invalid-json`)
	}))
	defer server.Close()

	oldClient := httpClient
	oldURL := jokeAPIURL
	httpClient = server.Client()
	jokeAPIURL = server.URL
	t.Cleanup(func() {
		httpClient = oldClient
		jokeAPIURL = oldURL
	})

	jokes := fetchBatchJokes()
	if len(jokes) != 1 || jokes[0] != fallbackJoke {
		t.Fatalf("expected single fallback joke, got %v", jokes)
	}
}

func TestFetchBatchJokesParsesMultipleJokes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := batchJokeResponse{Jokes: []jokeResponse{
			{Type: "single", Joke: "Joke one"},
			{Type: "twopart", Setup: "Why?", Delivery: "Because!"},
		}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldClient := httpClient
	oldURL := jokeAPIURL
	httpClient = server.Client()
	jokeAPIURL = server.URL
	t.Cleanup(func() {
		httpClient = oldClient
		jokeAPIURL = oldURL
	})

	jokes := fetchBatchJokes()
	if len(jokes) != 2 {
		t.Fatalf("expected 2 jokes, got %d: %v", len(jokes), jokes)
	}
	if jokes[0] != "Joke one" {
		t.Fatalf("unexpected first joke: %q", jokes[0])
	}
	if !strings.Contains(jokes[1], "Why?") || !strings.Contains(jokes[1], "Because!") {
		t.Fatalf("unexpected second joke: %q", jokes[1])
	}
}
