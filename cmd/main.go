package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/msafwankarim/quotingo/internal/repository"
)

// Author holds a team member's name and registration number.
type Author struct {
	Name  string
	RegNo string
}

var (
	version = "1.0.0"
	authors = []Author{
		{Name: "Muhammad Safwan Karim", RegNo: "537263"},
		{Name: "Muhammad Jahanzaib", RegNo: "537261"},
		{Name: "Muhamaad Mueed", RegNo: "537259"},
		{Name: "Waseem Gul", RegNo: "537276"},
		{Name: "Muneeb Baig", RegNo: "537241"},
	}

	jokeCache = &repository.JokeQueue{}

	tmpl *template.Template
)

type pageData struct {
	Message string
	Version string
	Authors []Author
	Joke    repository.JokeItem
}

func main() {
	tmpl = template.Must(template.ParseFiles("templates/index.html"))

	// Pre-fill the joke cache synchronously so the first request has a joke ready.
	jokeCache.Refill()

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

	data := pageData{
		Message: "Hello from Go!",
		Version: version,
		Authors: authors,
		Joke:    jokeCache.Next(),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "unable to render page", http.StatusInternalServerError)
	}
}
