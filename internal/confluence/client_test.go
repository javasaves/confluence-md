package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/javasaves/confluence-md/internal/confluence/model"
)

func TestFindPageIDByTitle(t *testing.T) {
	tests := []struct {
		name     string
		spaceKey string
		title    string
		results  []model.ConfluenceAPIPage
		wantID   string
		wantErr  string
	}{
		{
			name:     "single match",
			spaceKey: "TEAM",
			title:    "My Page",
			results:  []model.ConfluenceAPIPage{{ID: "12345", Title: "My Page"}},
			wantID:   "12345",
		},
		{
			name:    "no match",
			title:   "Missing",
			results: nil,
			wantErr: `page not found for title "Missing"`,
		},
		{
			name:    "multiple matches",
			title:   "Duplicate",
			results: []model.ConfluenceAPIPage{{ID: "1"}, {ID: "2"}},
			wantErr: `found 2 pages with title "Duplicate"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery url.Values
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(model.ConfluenceSearchResult{Results: tt.results})
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL, AuthConfig{Secret: "token"})
			id, err := client.FindPageIDByTitle(tt.spaceKey, tt.title)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Fatalf("got page ID %q, want %q", id, tt.wantID)
			}
			if gotQuery.Get("type") != "page" {
				t.Fatalf("expected type=page query param, got %q", gotQuery.Get("type"))
			}
			if gotQuery.Get("title") != tt.title {
				t.Fatalf("expected title query param %q, got %q", tt.title, gotQuery.Get("title"))
			}
			if tt.spaceKey != "" && gotQuery.Get("spaceKey") != tt.spaceKey {
				t.Fatalf("expected spaceKey %q, got %q", tt.spaceKey, gotQuery.Get("spaceKey"))
			}
			if tt.spaceKey == "" && gotQuery.Get("spaceKey") != "" {
				t.Fatalf("expected empty spaceKey, got %q", gotQuery.Get("spaceKey"))
			}
		})
	}
}

func TestFindPageIDByTitleEncodesSpecialCharacters(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(model.ConfluenceSearchResult{
			Results: []model.ConfluenceAPIPage{{ID: "99", Title: "A & B"}},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, AuthConfig{Secret: "token"})
	id, err := client.FindPageIDByTitle("", "A & B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "99" {
		t.Fatalf("got page ID %q, want %q", id, "99")
	}
	if gotQuery.Get("title") != "A & B" {
		t.Fatalf("expected decoded title in query %q, got %q", "A & B", gotQuery.Get("title"))
	}
}
