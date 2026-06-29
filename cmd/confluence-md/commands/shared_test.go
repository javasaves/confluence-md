package commands

import "testing"

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
