package model

import (
	"strings"
	"testing"
	"time"

	"github.com/javasaves/confluence-md/internal/confluence/model"
)

func TestMarkdownDocumentWithFrontmatter(t *testing.T) {
	doc := &MarkdownDocument{
		Frontmatter: Frontmatter{
			Title:  "Sample",
			Author: "Author",
			Date:   time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			Labels: []string{"one", "two"},
			Confluence: ConfluenceRef{
				PageID:   "123",
				SpaceKey: "SPACE",
				Version:  5,
				URL:      "https://example/spaces/SPACE/pages/123/Sample",
			},
			Custom: map[string]any{"custom": "value"},
		},
		Content: "Body",
	}

	out, err := doc.WithFrontmatter()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectations := []string{
		"title: \"Sample\"",
		"author: \"Author\"",
		"date: \"2024-01-02T03:04:05Z\"",
		"- \"one\"",
		"pageId: \"123\"",
		"custom: value",
		"Body",
	}

	for _, expect := range expectations {
		if !strings.Contains(out, expect) {
			t.Fatalf("expected output to contain %q, got %q", expect, out)
		}
	}
}

func TestNewMarkdownDocument(t *testing.T) {
	page := &model.ConfluencePage{
		ID:       "123",
		Title:    "Sample Page",
		SpaceKey: "SPACE",
		Version:  2,
		Content: model.ConfluenceContent{
			Storage: model.ContentStorage{Value: "<p>content</p>"},
		},
		Metadata: model.ConfluenceMetadata{Labels: []model.Label{{Name: "label"}}},
		CreatedBy: model.User{
			DisplayName: "Author",
		},
		UpdatedAt: time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	doc, err := NewMarkdownDocument(page, "https://example.atlassian.net/wiki", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Frontmatter.Title != "Sample Page" {
		t.Fatalf("unexpected title: %s", doc.Frontmatter.Title)
	}
	if doc.Frontmatter.Author != "Author" {
		t.Fatalf("unexpected author: %s", doc.Frontmatter.Author)
	}
	if doc.Frontmatter.Confluence.URL != "https://example.atlassian.net/wiki/spaces/SPACE/pages/123/Sample Page" {
		t.Fatalf("unexpected URL: %s", doc.Frontmatter.Confluence.URL)
	}
	if len(doc.Frontmatter.Labels) != 1 || doc.Frontmatter.Labels[0] != "label" {
		t.Fatalf("unexpected labels: %#v", doc.Frontmatter.Labels)
	}
}

func TestNewMarkdownDocumentUsesSourcePageURLVerbatim(t *testing.T) {
	page := &model.ConfluencePage{
		ID:       "12345",
		Title:    "API Title",
		SpaceKey: "TEAM",
		Version:  1,
		Content: model.ConfluenceContent{
			Storage: model.ContentStorage{Value: "<p>content</p>"},
		},
		CreatedBy: model.User{DisplayName: "Author"},
		UpdatedAt: time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	sourceURL := "https://wiki.company.local/spaces/TEAM/pages/12345/My+Original+Title"
	doc, err := NewMarkdownDocument(page, "https://wiki.company.local", sourceURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Frontmatter.Confluence.URL != sourceURL {
		t.Fatalf("expected source URL preserved, got %q want %q", doc.Frontmatter.Confluence.URL, sourceURL)
	}
}

func TestNewMarkdownDocumentUsesWebUIPathWhenNoSourceURL(t *testing.T) {
	page := &model.ConfluencePage{
		ID:        "12345",
		Title:     "API Title",
		SpaceKey:  "TEAM",
		Version:   1,
		WebUIPath: "/spaces/TEAM/pages/12345/My+Original+Title",
		Content: model.ConfluenceContent{
			Storage: model.ContentStorage{Value: "<p>content</p>"},
		},
		CreatedBy: model.User{DisplayName: "Author"},
		UpdatedAt: time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	doc, err := NewMarkdownDocument(page, "https://wiki.company.local", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://wiki.company.local/spaces/TEAM/pages/12345/My+Original+Title"
	if doc.Frontmatter.Confluence.URL != want {
		t.Fatalf("expected webui URL %q, got %q", want, doc.Frontmatter.Confluence.URL)
	}
}
