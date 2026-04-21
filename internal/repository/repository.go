package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	cacheSize    = 20
	fallbackItem = JokeItem{Setup: "Why do programmers prefer dark mode? Because light attracts bugs."}
	jokeAPIURL   = fmt.Sprintf("https://v2.jokeapi.dev/joke/Any?blacklistFlags=explicit&amount=%d", cacheSize)
	httpClient   = &http.Client{Timeout: 5 * time.Second}
)

type jokeResponse struct {
	Type     string `json:"type"`
	Joke     string `json:"joke"`
	Setup    string `json:"setup"`
	Delivery string `json:"delivery"`
}

type batchJokeResponse struct {
	Jokes []jokeResponse `json:"jokes"`
}

// JokeItem holds a single joke. Single jokes only populate Setup.
// Two-part jokes populate both Setup (the question) and Delivery (the punchline).
type JokeItem struct {
	Setup    string
	Delivery string
	TwoPart  bool
}

// jokeQueue is a thread-safe queue that pre-fetches jokes in bulk and serves
// them one at a time. When the queue is drained, it triggers an async refill.
type JokeQueue struct {
	Mutex     sync.Mutex
	items     []JokeItem
	refilling bool
}

// next dequeues and returns the next cached joke. When the last joke is
// dequeued it kicks off a background refill so the next batch is ready soon.
// Returns fallbackItem if the queue is currently empty (during a refill).
func (q *JokeQueue) Next() JokeItem {
	q.Mutex.Lock()

	if len(q.items) == 0 {
		needRefill := !q.refilling
		if needRefill {
			q.refilling = true
		}
		q.Mutex.Unlock()
		if needRefill {
			go q.Refill()
		}
		return fallbackItem
	}

	item := q.items[0]
	q.items = q.items[1:]

	needRefill := len(q.items) == 0 && !q.refilling
	if needRefill {
		q.refilling = true
	}
	q.Mutex.Unlock()

	if needRefill {
		go q.Refill()
	}
	return item
}

// Refill fetches a fresh batch of jokes from the API and replaces the queue,
// discarding any remaining stale entries.
func (q *JokeQueue) Refill() {
	fresh := fetchBatchJokes()

	q.Mutex.Lock()
	q.items = fresh
	q.refilling = false
	q.Mutex.Unlock()

	log.Printf("joke cache refilled with %d jokes", len(fresh))
}

// fetchBatchJokes calls the JokeAPI batch endpoint and returns the parsed
// jokes as JokeItems. Falls back to a single fallbackItem on any error.
func fetchBatchJokes() []JokeItem {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jokeAPIURL, nil)
	if err != nil {
		return []JokeItem{fallbackItem}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return []JokeItem{fallbackItem}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []JokeItem{fallbackItem}
	}

	var batch batchJokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return []JokeItem{fallbackItem}
	}

	var result []JokeItem
	for _, j := range batch.Jokes {
		if item, ok := toJokeItem(j); ok {
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return []JokeItem{fallbackItem}
	}
	return result
}

// toJokeItem converts a raw API response into a JokeItem.
// Returns false if the joke has no usable text.
func toJokeItem(j jokeResponse) (JokeItem, bool) {
	switch strings.ToLower(j.Type) {
	case "single":
		text := strings.TrimSpace(j.Joke)
		if text != "" {
			return JokeItem{Setup: text}, true
		}
	case "twopart":
		setup := strings.TrimSpace(j.Setup)
		delivery := strings.TrimSpace(j.Delivery)
		if setup != "" && delivery != "" {
			return JokeItem{Setup: setup, Delivery: delivery, TwoPart: true}, true
		}
	}
	return JokeItem{}, false
}
