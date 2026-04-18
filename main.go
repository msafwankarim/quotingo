package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const cacheSize = 10

var (
	version = "dev"
	authors = []string{
		"Muhammad Safwan Karim",
		"Muhammad Jahanzaib",
		"Muhamaad Mueed",
		"Waseem Gul",
		"Baig",
	}
	httpClient   = &http.Client{Timeout: 5 * time.Second}
	jokeAPIURL   = fmt.Sprintf("https://v2.jokeapi.dev/joke/Any?blacklistFlags=explicit&amount=%d", cacheSize)
	fallbackJoke = "Why do programmers prefer dark mode? Because light attracts bugs."

	jokeCache = &jokeQueue{}
)

// jokeQueue is a thread-safe queue that pre-fetches jokes in bulk and serves
// them one at a time. When the queue is drained, it triggers an async refill.
type jokeQueue struct {
	mu        sync.Mutex
	items     []string
	refilling bool
}

// next dequeues and returns the next cached joke. When the last joke is
// dequeued it kicks off a background refill so the next batch is ready soon.
// Returns fallbackJoke if the queue is currently empty (during a refill).
func (q *jokeQueue) next() string {
	q.mu.Lock()

	if len(q.items) == 0 {
		needRefill := !q.refilling
		if needRefill {
			q.refilling = true
		}
		q.mu.Unlock()
		if needRefill {
			go q.refill()
		}
		return fallbackJoke
	}

	joke := q.items[0]
	q.items = q.items[1:]

	needRefill := len(q.items) == 0 && !q.refilling
	if needRefill {
		q.refilling = true
	}
	q.mu.Unlock()

	if needRefill {
		go q.refill()
	}
	return joke
}

// refill fetches a fresh batch of jokes from the API and replaces the queue,
// discarding any remaining stale entries.
func (q *jokeQueue) refill() {
	fresh := fetchBatchJokes()

	q.mu.Lock()
	q.items = fresh
	q.refilling = false
	q.mu.Unlock()

	log.Printf("joke cache refilled with %d jokes", len(fresh))
}

type pageData struct {
	Message string
	Version string
	Authors []string
	Joke    string
}

type jokeResponse struct {
	Type     string `json:"type"`
	Joke     string `json:"joke"`
	Setup    string `json:"setup"`
	Delivery string `json:"delivery"`
}

type batchJokeResponse struct {
	Jokes []jokeResponse `json:"jokes"`
}

func main() {
	// Pre-fill the joke cache synchronously so the first request has a joke ready.
	jokeCache.refill()

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("go-hello running on :%s (version=%s)", port, version)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "unable to render page", http.StatusInternalServerError)
		return
	}

	data := pageData{
		Message: "Hello from Go!",
		Version: version,
		Authors: authors,
		Joke:    jokeCache.next(),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "unable to render page", http.StatusInternalServerError)
	}
}

// fetchBatchJokes calls the JokeAPI batch endpoint and returns the parsed
// jokes as plain strings. Falls back to a single fallbackJoke on any error.
func fetchBatchJokes() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jokeAPIURL, nil)
	if err != nil {
		return []string{fallbackJoke}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return []string{fallbackJoke}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{fallbackJoke}
	}

	var batch batchJokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return []string{fallbackJoke}
	}

	var result []string
	for _, j := range batch.Jokes {
		if text := jokeText(j); text != "" {
			result = append(result, text)
		}
	}

	if len(result) == 0 {
		return []string{fallbackJoke}
	}
	return result
}

// jokeText extracts a displayable string from a single joke payload.
func jokeText(j jokeResponse) string {
	switch strings.ToLower(j.Type) {
	case "single":
		return strings.TrimSpace(j.Joke)
	case "twopart":
		setup := strings.TrimSpace(j.Setup)
		delivery := strings.TrimSpace(j.Delivery)
		if setup != "" && delivery != "" {
			return setup + " — " + delivery
		}
	}
	return ""
}
