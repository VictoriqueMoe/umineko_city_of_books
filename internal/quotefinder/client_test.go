package quotefinder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCharacters_MainAndAdditional(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ciconia/characters" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"characters": {"miyao": "Miyao", "lingji": "Lingji"},
			"additional": {"narrator": "Narrator", "keropoyo": "Keropoyo"}
		}`)
	}))
	defer srv.Close()

	c := NewClientWithBaseURL(srv.URL)
	chars, err := c.ListCharacters(SeriesCiconia)
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}

	groups := map[string]string{}
	for i := 0; i < len(chars); i++ {
		groups[chars[i].ID] = chars[i].Group
	}

	if got := groups["miyao"]; got != "main" {
		t.Errorf(`chars["miyao"].Group = %q, want "main"`, got)
	}
	if got := groups["lingji"]; got != "main" {
		t.Errorf(`chars["lingji"].Group = %q, want "main"`, got)
	}
	if got := groups["narrator"]; got != "additional" {
		t.Errorf(`chars["narrator"].Group = %q, want "additional"`, got)
	}
	if got := groups["keropoyo"]; got != "additional" {
		t.Errorf(`chars["keropoyo"].Group = %q, want "additional"`, got)
	}
	if len(chars) != 4 {
		t.Errorf("len(chars) = %d, want 4", len(chars))
	}
}

func TestListCharacters_OnlyMain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"characters": {"beato": "Beatrice"}}`)
	}))
	defer srv.Close()

	c := NewClientWithBaseURL(srv.URL)
	chars, err := c.ListCharacters(SeriesUmineko)
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}

	if len(chars) != 1 {
		t.Fatalf("len(chars) = %d, want 1", len(chars))
	}
	if chars[0].Group != "main" {
		t.Errorf("chars[0].Group = %q, want main", chars[0].Group)
	}
}

func TestListCharacters_InvalidSeries(t *testing.T) {
	c := NewClientWithBaseURL("http://never-called.invalid")
	if _, err := c.ListCharacters(Series("roseguns")); err == nil {
		t.Fatal("expected error for unsupported series, got nil")
	}
}

func TestListCharacters_CachesResult(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"characters": {"rika": "Rika"}}`)
	}))
	defer srv.Close()

	c := NewClientWithBaseURL(srv.URL)
	for i := 0; i < 3; i++ {
		if _, err := c.ListCharacters(SeriesHigurashi); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if hits != 1 {
		t.Fatalf("expected 1 upstream hit (rest cached), got %d", hits)
	}
}
