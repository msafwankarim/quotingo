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

const cacheSize = 20

var (
	version = "1.0.0"
	authors = []string{
		"Muhammad Safwan Karim",
		"Muhammad Jahanzaib",
		"Muhamaad Mueed",
		"Waseem Gul",
		"Baig",
	}
	httpClient   = &http.Client{Timeout: 5 * time.Second}
	jokeAPIURL   = fmt.Sprintf("https://v2.jokeapi.dev/joke/Any?blacklistFlags=explicit&amount=%d", cacheSize)
	fallbackItem = jokeItem{Setup: "Why do programmers prefer dark mode? Because light attracts bugs."}

	jokeCache = &jokeQueue{}
)

// jokeItem holds a single joke. Single jokes only populate Setup.
// Two-part jokes populate both Setup (the question) and Delivery (the punchline).
type jokeItem struct {
	Setup    string
	Delivery string
	TwoPart  bool
}

// jokeQueue is a thread-safe queue that pre-fetches jokes in bulk and serves
// them one at a time. When the queue is drained, it triggers an async refill.
type jokeQueue struct {
	mu        sync.Mutex
	items     []jokeItem
	refilling bool
}

// next dequeues and returns the next cached joke. When the last joke is
// dequeued it kicks off a background refill so the next batch is ready soon.
// Returns fallbackItem if the queue is currently empty (during a refill).
func (q *jokeQueue) next() jokeItem {
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
		return fallbackItem
	}

	item := q.items[0]
	q.items = q.items[1:]

	needRefill := len(q.items) == 0 && !q.refilling
	if needRefill {
		q.refilling = true
	}
	q.mu.Unlock()

	if needRefill {
		go q.refill()
	}
	return item
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
	Joke    jokeItem
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

	log.Printf("quotingo running on :%s (version=%s)", port, version)
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
// jokes as jokeItems. Falls back to a single fallbackItem on any error.
func fetchBatchJokes() []jokeItem {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jokeAPIURL, nil)
	if err != nil {
		return []jokeItem{fallbackItem}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return []jokeItem{fallbackItem}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []jokeItem{fallbackItem}
	}

	var batch batchJokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return []jokeItem{fallbackItem}
	}

	var result []jokeItem
	for _, j := range batch.Jokes {
		if item, ok := toJokeItem(j); ok {
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return []jokeItem{fallbackItem}
	}
	return result
}

// toJokeItem converts a raw API response into a jokeItem.
// Returns false if the joke has no usable text.
func toJokeItem(j jokeResponse) (jokeItem, bool) {
	switch strings.ToLower(j.Type) {
	case "single":
		text := strings.TrimSpace(j.Joke)
		if text != "" {
			return jokeItem{Setup: text}, true
		}
	case "twopart":
		setup := strings.TrimSpace(j.Setup)
		delivery := strings.TrimSpace(j.Delivery)
		if setup != "" && delivery != "" {
			return jokeItem{Setup: setup, Delivery: delivery, TwoPart: true}, true
		}
	}
	return jokeItem{}, false
}
