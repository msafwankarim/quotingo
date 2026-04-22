package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Minimal template so homeHandler can execute without loading the real file.
	tmpl = template.Must(template.New("test").Parse(
		"{{.Message}}|{{.Version}}|{{range .Authors}}{{.Name}},{{end}}|{{.Joke.Setup}}",
	))
	os.Exit(m.Run())
}

func TestHomeHandler_RootReturns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	homeHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello from Go!") {
		t.Errorf("body missing message: %q", body)
	}
	if !strings.Contains(body, version) {
		t.Errorf("body missing version: %q", body)
	}
}

func TestHomeHandler_NonRootReturns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()

	homeHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
