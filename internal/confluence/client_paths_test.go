package confluence

import (
	"testing"

	"github.com/javasaves/confluence-md/internal/confluence/model"
)

func TestSplitBaseURL(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantOriginURL  string
		wantBasePath   string
	}{
		{
			name:          "root context",
			input:         "https://wiki.company.local",
			wantOriginURL: "https://wiki.company.local",
			wantBasePath:  "",
		},
		{
			name:          "wiki context",
			input:         "https://example.atlassian.net/wiki",
			wantOriginURL: "https://example.atlassian.net",
			wantBasePath:  "/wiki",
		},
		{
			name:          "custom context",
			input:         "https://wiki.company.local/confluence",
			wantOriginURL: "https://wiki.company.local",
			wantBasePath:  "/confluence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOriginURL, gotBasePath := splitBaseURL(tt.input)
			if gotOriginURL != tt.wantOriginURL {
				t.Fatalf("unexpected origin URL %q, want %q", gotOriginURL, tt.wantOriginURL)
			}
			if gotBasePath != tt.wantBasePath {
				t.Fatalf("unexpected base path %q, want %q", gotBasePath, tt.wantBasePath)
			}
		})
	}
}

func TestNormalizeDownloadLinkUsesDerivedBasePath(t *testing.T) {
	tests := []struct {
		name     string
		client   *client
		link     string
		wantURL  string
	}{
		{
			name: "root context",
			client: &client{
				originURL: "https://wiki.company.local",
			},
			link:    "/download/attachments/123/file.png",
			wantURL: "https://wiki.company.local/download/attachments/123/file.png",
		},
		{
			name: "wiki context",
			client: &client{
				originURL: "https://example.atlassian.net",
				basePath:  "/wiki",
			},
			link:    "/download/attachments/123/file.png",
			wantURL: "https://example.atlassian.net/wiki/download/attachments/123/file.png",
		},
		{
			name: "custom context already present",
			client: &client{
				originURL: "https://wiki.company.local",
				basePath:  "/confluence",
			},
			link:    "/confluence/download/attachments/123/file.png",
			wantURL: "https://wiki.company.local/confluence/download/attachments/123/file.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.client.normalizeDownloadLink(tt.link)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantURL {
				t.Fatalf("unexpected URL %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestAttachmentRESTDownloadURLUsesBasePath(t *testing.T) {
	c := &client{
		originURL: "https://wiki.company.local",
		basePath:  "/confluence",
	}

	attachment := &model.ConfluenceAttachment{
		ID:           "987",
		DownloadLink: "/download/attachments/123/file.png",
	}

	got, ok := c.attachmentRESTDownloadURL(attachment)
	if !ok {
		t.Fatal("expected fallback attachment URL to be available")
	}

	want := "https://wiki.company.local/confluence/rest/api/content/123/child/attachment/987/download"
	if got != want {
		t.Fatalf("unexpected attachment REST URL %q, want %q", got, want)
	}
}
