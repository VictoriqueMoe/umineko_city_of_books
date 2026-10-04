package quotefinder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/cache/engines"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewTestServer(t, handler)
	httpClient := srv.Client()

	c := NewClientWithBaseURL(srv.URL, cache.NewManager(engines.NewInMemory(0)))
	c.http = httpClient

	return c
}

func TestListCharacters_MainAndAdditional(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ciconia/characters" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"characters": {"miyao": "Miyao", "lingji": "Lingji"},
			"additional": {"narrator": "Narrator", "keropoyo": "Keropoyo"}
		}`)
	})

	chars, err := c.ListCharacters(t.Context(), SeriesCiconia)
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}

	groups := map[string]string{}
	for i := range chars {
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
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"characters": {"beato": "Beatrice"}}`)
	})

	chars, err := c.ListCharacters(t.Context(), SeriesUmineko)
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
	c := NewClientWithBaseURL("http://never-called.invalid", nil)
	if _, err := c.ListCharacters(t.Context(), "roseguns"); err == nil {
		t.Fatal("expected error for unsupported series, got nil")
	}
}

func TestListCharacters_CachesResult(t *testing.T) {
	var hits int

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"characters": {"rika": "Rika"}}`)
	})

	for i := range 3 {
		if _, err := c.ListCharacters(t.Context(), SeriesHigurashi); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if hits != 1 {
		t.Fatalf("expected 1 upstream hit (rest cached), got %d", hits)
	}
}

func TestGetQuote_CachesResult(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		fetch func(c *Client) (*Quote, error)
	}{
		{
			name: "by audio id",
			path: "/umineko/quote/10100001",
			fetch: func(c *Client) (*Quote, error) {
				return c.GetByAudioID(t.Context(), SeriesUmineko, "10100001, 10100002")
			},
		},
		{
			name: "by index",
			path: "/umineko/quote/index/42",
			fetch: func(c *Client) (*Quote, error) {
				return c.GetByIndex(t.Context(), SeriesUmineko, 42)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var hits int
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				hits++
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"hasRedTruth": true}`)
			})

			// when
			var q *Quote
			for i := range 3 {
				got, err := tc.fetch(c)
				if err != nil {
					t.Fatalf("call %d: %v", i, err)
				}
				q = got
			}

			// then
			if hits != 1 {
				t.Fatalf("expected 1 upstream hit (rest cached), got %d", hits)
			}
			if q == nil || !q.HasRedTruth {
				t.Fatalf("quote = %+v, want HasRedTruth", q)
			}
		})
	}
}

func TestGetQuote_UpstreamStatus(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		wantErr  bool
		wantHits int
	}{
		{name: "not found is cached as no quote", status: http.StatusNotFound, wantErr: false, wantHits: 1},
		{name: "server error is returned and not cached", status: http.StatusInternalServerError, wantErr: true, wantHits: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var hits int
			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				hits++
				w.WriteHeader(tc.status)
			})

			// when
			var errs []error
			for range 2 {
				q, err := c.GetByIndex(t.Context(), SeriesUmineko, 7)
				if q != nil {
					t.Fatalf("quote = %+v, want nil", q)
				}
				errs = append(errs, err)
			}

			// then
			for i, err := range errs {
				if (err != nil) != tc.wantErr {
					t.Fatalf("call %d: err = %v, wantErr %v", i, err, tc.wantErr)
				}
			}
			if hits != tc.wantHits {
				t.Fatalf("upstream hits = %d, want %d", hits, tc.wantHits)
			}
		})
	}
}
