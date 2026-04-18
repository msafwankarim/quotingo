package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	version = "dev"
	authors = []string{
		"Muhammad Safwan Karim",
		"Muhammad Jahanzaib",
		"Muhamaad Mueed",
		"Waseem Gul",
		"Baig",
	}
	httpClient   = &http.Client{Timeout: 3 * time.Second}
	jokeAPIURL   = "https://v2.jokeapi.dev/joke/Any?blacklistFlags=explicit"
	fallbackJoke = "Why do programmers prefer dark mode? Because light attracts bugs."
)

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

func main() {
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
		Joke:    fetchJoke(r.Context()),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "unable to render page", http.StatusInternalServerError)
	}
}

func fetchJoke(parent context.Context) string {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jokeAPIURL, nil)
	if err != nil {
		return fallbackJoke
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fallbackJoke
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackJoke
	}

	var joke jokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&joke); err != nil {
		return fallbackJoke
	}

	switch strings.ToLower(joke.Type) {
	case "single":
		if strings.TrimSpace(joke.Joke) != "" {
			return joke.Joke
		}
	case "twopart":
		setup := strings.TrimSpace(joke.Setup)
		delivery := strings.TrimSpace(joke.Delivery)
		if setup != "" && delivery != "" {
			return setup + " — " + delivery
		}
	}

	return fallbackJoke
}
