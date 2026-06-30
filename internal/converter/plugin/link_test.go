package plugin

import "testing"

func TestEscapeMarkdownLinkText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "brackets and underscores",
			in:   "[Core][SQL] op.program_get_by_name",
			want: `\[Core\]\[SQL\] op.program\_get\_by\_name`,
		},
		{
			name: "emphasis markers",
			in:   "*bold* and _italic_",
			want: `\*bold\* and \_italic\_`,
		},
		{
			name: "backtick code",
			in:   "use `code` here",
			want: "use \\`code\\` here",
		},
		{
			name: "backslash",
			in:   `path\file`,
			want: `path\\file`,
		},
		{
			name: "plain text unchanged",
			in:   "Hello world 123",
			want: "Hello world 123",
		},
		{
			name: "cyrillic unchanged",
			in:   "[ХП] название",
			want: `\[ХП\] название`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeMarkdownLinkText(tt.in); got != tt.want {
				t.Fatalf("escapeMarkdownLinkText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEscapeMarkdownLinkURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "parentheses in query",
			in:   "/pages/viewpage.action?pageId=1&foo=(bar)",
			want: `/pages/viewpage.action?pageId=1&foo=\(bar\)`,
		},
		{
			name: "plain path unchanged",
			in:   "/spaces/CORE/pages/123/Title",
			want: "/spaces/CORE/pages/123/Title",
		},
		{
			name: "backslash",
			in:   `https://example.com/path\segment`,
			want: `https://example.com/path\\segment`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeMarkdownLinkURL(tt.in); got != tt.want {
				t.Fatalf("escapeMarkdownLinkURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractHrefFromAttrs(t *testing.T) {
	tests := []struct {
		name  string
		attrs string
		want  string
	}{
		{
			name:  "double quoted href",
			attrs: ` class="confluence-link" href="/pages/viewpage.action?pageId=42" data-linked-resource-id="42"`,
			want:  "/pages/viewpage.action?pageId=42",
		},
		{
			name:  "single quoted href",
			attrs: ` class='confluence-link' href='/display/CORE/My+Page'`,
			want:  "/display/CORE/My+Page",
		},
		{
			name:  "missing href",
			attrs: ` class="confluence-link"`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractHrefFromAttrs(tt.attrs); got != tt.want {
				t.Fatalf("extractHrefFromAttrs(%q) = %q, want %q", tt.attrs, got, tt.want)
			}
		})
	}
}
