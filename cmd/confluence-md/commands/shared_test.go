package commands

import (
	"fmt"
	"strings"
	"testing"

	mock_confluence "github.com/javasaves/confluence-md/internal/confluence/mock"
	confluenceModel "github.com/javasaves/confluence-md/internal/confluence/model"
	"go.uber.org/mock/gomock"
)

func TestURLToPageInfoDerivesBaseURLFromPagePath(t *testing.T) {
	tests := []struct {
		name        string
		pageURL     string
		wantBaseURL string
		wantPageID  string
	}{
		{
			name:        "root context",
			pageURL:     "https://wiki.company.local/spaces/TEAM/pages/12345/Title",
			wantBaseURL: "https://wiki.company.local",
			wantPageID:  "12345",
		},
		{
			name:        "wiki context",
			pageURL:     "https://example.atlassian.net/wiki/spaces/TEAM/pages/12345/Title",
			wantBaseURL: "https://example.atlassian.net/wiki",
			wantPageID:  "12345",
		},
		{
			name:        "custom context",
			pageURL:     "https://wiki.company.local/confluence/spaces/TEAM/pages/12345/Title",
			wantBaseURL: "https://wiki.company.local/confluence",
			wantPageID:  "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := urlToPageInfo(tt.pageURL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.BaseURL != tt.wantBaseURL {
				t.Fatalf("unexpected base URL %q, want %q", info.BaseURL, tt.wantBaseURL)
			}

			if info.PageID != tt.wantPageID {
				t.Fatalf("unexpected page ID %q, want %q", info.PageID, tt.wantPageID)
			}
		})
	}
}

func TestURLToPageInfoSpacesNonNumericPageID(t *testing.T) {
	_, err := urlToPageInfo("https://wiki.company.local/spaces/TEAM/pages/not-a-id/Title")
	if err == nil {
		t.Fatal("expected parse error for non-numeric page ID in spaces URL")
	}
	if !strings.Contains(err.Error(), errCouldNotExtractPageID) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLToPageInfoDisplayURL(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/wiki/display/SPACE/My+Page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.BaseURL != "https://wiki.company.local/wiki" {
		t.Fatalf("unexpected base URL %q", info.BaseURL)
	}
	if info.SpaceKey != "SPACE" {
		t.Fatalf("unexpected space key %q", info.SpaceKey)
	}
	if info.Title != "My Page" {
		t.Fatalf("unexpected title %q, want %q", info.Title, "My Page")
	}
	if info.PageID != "" {
		t.Fatalf("expected empty page ID for display URL, got %q", info.PageID)
	}
}

func TestURLToPageInfoDisplayURLEncodedTitle(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/display/SPACE/My%20Page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Title != "My Page" {
		t.Fatalf("unexpected title %q, want %q", info.Title, "My Page")
	}
}

func TestURLToPageInfoViewpageWithTitleQuery(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/pages/viewpage.action?title=My+Page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Title != "My Page" {
		t.Fatalf("unexpected title %q, want %q", info.Title, "My Page")
	}
	if info.Title == "viewpage.action" {
		t.Fatal("title must not be taken from path segment viewpage.action")
	}
	if info.PageID != "" {
		t.Fatalf("expected empty page ID, got %q", info.PageID)
	}
}

func TestURLToPageInfoViewpageWithPageID(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/pages/viewpage.action?pageId=12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PageID != "12345" {
		t.Fatalf("unexpected page ID %q", info.PageID)
	}
}

func TestURLToPageInfoViewpageWithInvalidPageID(t *testing.T) {
	_, err := urlToPageInfo("https://wiki.company.local/pages/viewpage.action?pageId=abc")
	if err == nil {
		t.Fatal("expected parse error for non-numeric pageId query param")
	}
}

func TestURLToPageInfoViewpageWithBasePath(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/wiki/pages/viewpage.action?pageId=12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.BaseURL != "https://wiki.company.local/wiki" {
		t.Fatalf("unexpected base URL %q, want %q", info.BaseURL, "https://wiki.company.local/wiki")
	}
	if info.PageID != "12345" {
		t.Fatalf("unexpected page ID %q", info.PageID)
	}
}

func TestURLToPageInfoViewpageUnderSpacesPath(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/wiki/spaces/TEAM/pages/viewpage.action?pageId=12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.BaseURL != "https://wiki.company.local/wiki" {
		t.Fatalf("unexpected base URL %q, want %q", info.BaseURL, "https://wiki.company.local/wiki")
	}
	if info.SpaceKey != "TEAM" {
		t.Fatalf("unexpected space key %q, want %q", info.SpaceKey, "TEAM")
	}
	if info.PageID != "12345" {
		t.Fatalf("unexpected page ID %q", info.PageID)
	}
}

func TestURLToPageInfoViewpageUnderSpacesWithTitle(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/wiki/spaces/TEAM/pages/viewpage.action?title=My+Page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SpaceKey != "TEAM" {
		t.Fatalf("unexpected space key %q, want %q", info.SpaceKey, "TEAM")
	}
	if info.Title != "My Page" {
		t.Fatalf("unexpected title %q", info.Title)
	}
}

func TestURLToPageInfoViewpageEmptyQuery(t *testing.T) {
	_, err := urlToPageInfo("https://wiki.company.local/pages/viewpage.action")
	if err == nil {
		t.Fatal("expected parse error for viewpage URL without pageId or title")
	}
	if !strings.Contains(err.Error(), errCouldNotExtractPageIDOrTitle) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLToPageInfoSpacesQueryPageIDOverridesPath(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/spaces/TEAM/pages/not-a-id/Title?pageId=12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PageID != "12345" {
		t.Fatalf("unexpected page ID %q, want %q", info.PageID, "12345")
	}
}

func TestURLToPageInfoDisplayWithQueryPageID(t *testing.T) {
	info, err := urlToPageInfo("https://wiki.company.local/display/SPACE/My+Page?pageId=99999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PageID != "99999" {
		t.Fatalf("unexpected page ID %q, want %q", info.PageID, "99999")
	}
	if info.Title != "My Page" {
		t.Fatalf("unexpected title %q", info.Title)
	}
}

func TestEnsurePageIDSkipsWhenPageIDPresent(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_confluence.NewMockClient(ctrl)

	info := confluenceModel.PageURLInfo{PageID: "12345", Title: "My Page"}
	got, err := ensurePageID(client, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PageID != "12345" {
		t.Fatalf("unexpected page ID %q", got.PageID)
	}
}

func TestEnsurePageIDResolvesByTitle(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_confluence.NewMockClient(ctrl)

	info := confluenceModel.PageURLInfo{
		SpaceKey: "SPACE",
		Title:    "My Page",
	}
	client.EXPECT().
		FindPageIDByTitle("SPACE", "My Page").
		Return("99999", nil)

	got, err := ensurePageID(client, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PageID != "99999" {
		t.Fatalf("unexpected page ID %q, want %q", got.PageID, "99999")
	}
}

func TestEnsurePageIDPropagatesLookupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_confluence.NewMockClient(ctrl)

	info := confluenceModel.PageURLInfo{
		SpaceKey: "SPACE",
		Title:    "Missing",
	}
	client.EXPECT().
		FindPageIDByTitle("SPACE", "Missing").
		Return("", fmt.Errorf(`page not found for title "Missing"`))

	_, err := ensurePageID(client, info)
	if err == nil {
		t.Fatal("expected lookup error")
	}
	if !strings.Contains(err.Error(), `page not found for title "Missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
