package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	confModel "github.com/javasaves/confluence-md/internal/confluence/model"
	convModel "github.com/javasaves/confluence-md/internal/converter/model"
	mock_attachments "github.com/javasaves/confluence-md/internal/converter/plugin/attachments/mock"
	gomock "go.uber.org/mock/gomock"
)

func TestConverterConvertPage(t *testing.T) {
	conv := NewConverter(nil)

	page := &confModel.ConfluencePage{
		ID:       "123",
		Title:    "Sample Page",
		SpaceKey: "SPACE",
		Version:  1,
		Content: confModel.ConfluenceContent{
			Storage: confModel.ContentStorage{
				Value: "<p>Hello World</p><ac:image ri:filename=\"diagram.png\"></ac:image>",
			},
		},
		Metadata: confModel.ConfluenceMetadata{
			Labels: []confModel.Label{{Name: "Label"}},
		},
		CreatedBy: confModel.User{DisplayName: "Author"},
		UpdatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	page.Content.Storage.Representation = "storage"
	page.CreatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	page.UpdatedBy = confModel.User{DisplayName: "Editor"}

	tests := []struct {
		name    string
		page    *confModel.ConfluencePage
		wantErr string
	}{
		{
			name: "success",
			page: page,
		},
		{
			name:    "invalid page",
			page:    &confModel.ConfluencePage{Title: "Missing ID", Content: confModel.ConfluenceContent{Storage: confModel.ContentStorage{Value: "<p>content</p>"}}, SpaceKey: "SPACE"},
			wantErr: "page ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := conv.ConvertPage(tt.page, "https://example.atlassian.net", ".", "")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				if doc != nil {
					t.Fatalf("expected nil doc, got %#v", doc)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc == nil {
				t.Fatal("expected document, got nil")
			}
			if !strings.Contains(doc.Content, "Hello World") {
				t.Fatalf("expected markdown content, got %q", doc.Content)
			}
		})
	}
}

func TestConverterConvertPageUsesBrowserPageLinks(t *testing.T) {
	conv := NewConverter(nil)
	page := &confModel.ConfluencePage{
		ID:       "123",
		Title:    "Sample Page",
		SpaceKey: "SPACE",
		Version:  1,
		Content: confModel.ConfluenceContent{
			Storage: confModel.ContentStorage{
				Value: `<p>See <ac:link><ri:content-entity ri:content-id="999"/><ac:plain-text-link-body>Target</ac:plain-text-link-body></ac:link></p>`,
			},
		},
		CreatedBy: confModel.User{DisplayName: "Author"},
		UpdatedBy: confModel.User{DisplayName: "Editor"},
	}
	page.Content.Storage.Representation = "storage"
	page.CreatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	page.UpdatedAt = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	doc, err := conv.ConvertPage(page, "https://wiki.example.com", ".", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "https://wiki.example.com/pages/viewpage.action?pageId=999"
	if !strings.Contains(doc.Content, want) {
		t.Fatalf("expected browser page link %q in %q", want, doc.Content)
	}
}

func TestConverterConvertPageDerivesJiraLinks(t *testing.T) {
	conv := NewConverter(nil)
	page := &confModel.ConfluencePage{
		ID:       "123",
		Title:    "Jira Page",
		SpaceKey: "SPACE",
		Version:  1,
		Content: confModel.ConfluenceContent{
			Storage: confModel.ContentStorage{
				Value: `<p>Before <ac:structured-macro ac:name="jira"><ac:parameter ac:name="key">ENG-123</ac:parameter></ac:structured-macro> after</p>`,
			},
		},
		CreatedBy: confModel.User{DisplayName: "Author"},
		UpdatedBy: confModel.User{DisplayName: "Editor"},
	}
	page.Content.Storage.Representation = "storage"
	page.CreatedAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	page.UpdatedAt = time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	doc, err := conv.ConvertPage(page, "https://confluence.example.com", ".", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Before [ENG-123](https://jira.example.com/browse/ENG-123) after"
	if !strings.Contains(doc.Content, want) {
		t.Fatalf("expected jira link %q in markdown, got %q", want, doc.Content)
	}
}

func TestConverterTableCellLinkInsideStrongAndSpanComplexCell(t *testing.T) {
	conv := NewConverter(nil)
	conv.confluenceBaseURL = "https://wiki.example.com"
	conv.plugin.SetBaseURL("https://wiki.example.com")
	conv.plugin.SetCurrentPage(&confModel.ConfluencePage{SpaceKey: "CORE"})

	html := `<table><tbody><tr><td><p>Line one</p><p><strong>option_program.op_program<span style="color:var(--ds-text-accent-blue-bolder,#09326c);">_get<ac:link><ri:page ri:content-title="[Core][SQL][Таблица] option_program"></ri:page></ac:link></span></strong></p></td></tr></tbody></table>`

	got, err := conv.convertHtml(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(got, "<ac:link>") {
		t.Fatalf("expected ac:link converted in complex table cell, got %q", got)
	}
	if !strings.Contains(got, "https://wiki.example.com/display/CORE/") {
		t.Fatalf("expected absolute page link, got %q", got)
	}
}

func TestConverterConvertHTMLConfluencePageLinksRegression(t *testing.T) {
	conv := NewConverter(nil)
	conv.confluenceBaseURL = "https://wiki.example.com"
	conv.plugin.SetBaseURL("https://wiki.example.com")
	conv.plugin.SetCurrentPage(&confModel.ConfluencePage{SpaceKey: "CORE"})

	html := `<ul>
<li><p>Вызвать процедуру … со следующими… <ac:link><ri:page ri:content-title="[Core][SQL][ХП] op.program_get_by_prog_name"/></ac:link></p></li>
</ul>
<table>
<tbody>
<tr>
<td><p><strong>option_program.op_program_get <ac:link><ri:page ri:content-title="[Core][SQL][Таблица] option_program"/></ac:link></strong></p></td>
</tr>
</tbody>
</table>`

	got, err := conv.convertHtml(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(got, "confluence://") {
		t.Fatalf("expected browser links, still have confluence:// scheme in %q", got)
	}
	if !strings.Contains(got, "https://wiki.example.com/display/") {
		t.Fatalf("expected display page links in %q", got)
	}
}

func TestFixMarkdownLinksPreservesConfluenceScheme(t *testing.T) {
	input := `See [Page](confluence://pageId/12345) for details`
	got := fixMarkdownLinks(input, "https://wiki.example.com")
	if got != input {
		t.Fatalf("fixMarkdownLinks() = %q, want unchanged %q", got, input)
	}
}

func TestFixMarkdownLinksPrefixesDisplayPath(t *testing.T) {
	input := `See [Page](/display/CORE/My+Page) for details`
	got := fixMarkdownLinks(input, "https://wiki.example.com")
	want := `See [Page](https://wiki.example.com/display/CORE/My+Page) for details`
	if got != want {
		t.Fatalf("fixMarkdownLinks() = %q, want %q", got, want)
	}
}

func TestConverterConvertHTMLConfluencePageLinkWithPlainTextBody(t *testing.T) {
	conv := NewConverter(nil)

	html := `<p><ac:link><ri:page ri:content-id="42" ri:content-title="fallback"/><ac:plain-text-link-body>Link [brackets] _text</ac:plain-text-link-body></ac:link></p>`

	got, err := conv.ConvertHTML(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `[Link \[brackets\] \_text](/pages/viewpage.action?pageId=42)`
	if !strings.Contains(got, want) {
		t.Fatalf("expected plain-text page link %q in %q", want, got)
	}
}

func TestConverterConvertHTMLPreservesInlineJiraSpacingWithoutBaseURL(t *testing.T) {
	conv := NewConverter(nil)

	got, err := conv.ConvertHTML(`<p>Before <ac:structured-macro ac:name="jira"><ac:parameter ac:name="key">ENG-123</ac:parameter></ac:structured-macro> after</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Before ENG-123 after"
	if got != want {
		t.Fatalf("expected inline jira markdown %q, got %q", want, got)
	}
}

func TestConverterConvertHTMLPreservesInlineUnsupportedMacroSpacing(t *testing.T) {
	conv := NewConverter(nil)

	got, err := conv.ConvertHTML(`<p>Before <ac:structured-macro ac:name="drawio"></ac:structured-macro> after</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Before **Unsupported macro:** `drawio` after"
	if got != want {
		t.Fatalf("expected inline unsupported macro markdown %q, got %q", want, got)
	}
}

func TestConverterConvertHTMLPreservesInlineJiraPunctuation(t *testing.T) {
	conv := NewConverter(nil)

	got, err := conv.ConvertHTML(`<p>Before (<ac:structured-macro ac:name="jira"><ac:parameter ac:name="key">ENG-123</ac:parameter></ac:structured-macro>), after</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Before (ENG-123), after"
	if got != want {
		t.Fatalf("expected inline jira markdown with punctuation %q, got %q", want, got)
	}
}

func TestConverterDownloadImages(t *testing.T) {
	data := []byte("image-bytes")
	attachment := &confModel.ConfluenceAttachment{Title: "diagram.png", MediaType: "image/png", FileSize: int64(len(data))}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockResolver := mock_attachments.NewMockResolver(ctrl)
	mockResolver.EXPECT().DownloadAttachment(gomock.Any(), "diagram.png", 0).Return(attachment, data, nil)

	conv := &Converter{
		imageFolder: "images",
		attachments: mockResolver,
	}

	doc := &convModel.MarkdownDocument{
		Images: []convModel.ImageRef{{
			FileName: "diagram.png",
		}},
	}

	page := &confModel.ConfluencePage{
		Attachments: []confModel.ConfluenceAttachment{{Title: "diagram.png"}},
	}

	tmpDir := t.TempDir()

	if err := conv.downloadImages(doc, page, tmpDir); err != nil {
		t.Fatalf("DownloadImages returned error: %v", err)
	}

	imagePath := filepath.Join(tmpDir, "images", "diagram.png")
	got, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("failed to read downloaded image: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("unexpected image content: %q", string(got))
	}
	if doc.Images[0].ContentType != "image/png" {
		t.Fatalf("expected content type image/png, got %q", doc.Images[0].ContentType)
	}
	if doc.Images[0].Size != int64(len(data)) {
		t.Fatalf("expected size %d, got %d", len(data), doc.Images[0].Size)
	}
}

func TestSaveMarkdownDocument(t *testing.T) {
	tmpDir := t.TempDir()
	doc := &convModel.MarkdownDocument{
		Content: "body",
		Frontmatter: convModel.Frontmatter{
			Title:  "Title",
			Author: "Author",
			Date:   time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			Confluence: convModel.ConfluenceRef{
				PageID:   "123",
				SpaceKey: "SPACE",
				Version:  1,
				URL:      "https://example.atlassian.net/wiki/pages/123",
			},
		},
	}

	plainPath := filepath.Join(tmpDir, "doc.md")
	if err := SaveMarkdownDocument(doc, plainPath, false); err != nil {
		t.Fatalf("SaveMarkdownDocument returned error: %v", err)
	}

	plainContent, err := os.ReadFile(plainPath)
	if err != nil {
		t.Fatalf("failed to read markdown file: %v", err)
	}
	if string(plainContent) != "body" {
		t.Fatalf("unexpected markdown content: %q", string(plainContent))
	}

	// Reset content and save with frontmatter
	doc.Content = "body"
	frontPath := filepath.Join(tmpDir, "doc-with-frontmatter.md")
	if err := SaveMarkdownDocument(doc, frontPath, true); err != nil {
		t.Fatalf("SaveMarkdownDocument with frontmatter returned error: %v", err)
	}

	frontContent, err := os.ReadFile(frontPath)
	if err != nil {
		t.Fatalf("failed to read frontmatter file: %v", err)
	}
	frontStr := string(frontContent)
	if !strings.HasPrefix(frontStr, "---\n") {
		t.Fatalf("expected frontmatter prefix, got %q", frontStr)
	}
	if !strings.Contains(frontStr, "title: \"Title\"") {
		t.Fatalf("expected title in frontmatter, got %q", frontStr)
	}
	if doc.Content != frontStr {
		t.Fatalf("expected document content updated with frontmatter")
	}
}

func TestConverterPostprocessMarkdown(t *testing.T) {
	conv := NewConverter(nil)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "collapse blank lines",
			input: "line1\n\n\nline2",
			want:  "line1\n\nline2",
		},
		{
			name:  "trim whitespace",
			input: "  content  \n\n",
			want:  "content",
		},
		{
			name:  "fix nested list spacing",
			input: "\n- item\n\n  - nested\n",
			want:  "- item\n  - nested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := conv.postprocessMarkdown(tt.input)
			if got != tt.want {
				t.Fatalf("postprocessMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConverterPreprocessCDATA(t *testing.T) {
	conv := NewConverter(nil, nil)
	input := "<![CDATA[<tag>&value]]>"
	got := conv.preprocessCDATA(input)
	if !strings.Contains(got, "<pre data-cdata='true'>") {
		t.Fatalf("expected pre block, got %q", got)
	}
	if strings.Contains(got, "<![CDATA[") {
		t.Fatalf("expected cdata markers removed, got %q", got)
	}
	if !strings.Contains(got, "&lt;tag&gt;") {
		t.Fatalf("expected html escaped content, got %q", got)
	}
}

func TestFixMarkdownLinks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		baseURL string
		want    string
	}{
		{
			name:  "root context internal without base",
			input: "See [Page](/spaces/SPACE/pages/12345/Some-Page) for details",
			want:  "See [Page](/spaces/SPACE/pages/12345/Some-Page) for details",
		},
		{
			name:  "viewpage action without base",
			input: "See [Page](/pages/viewpage.action?pageId=12345) for details",
			want:  "See [Page](/pages/viewpage.action?pageId=12345) for details",
		},
		{
			name:    "spaces path absolute",
			input:   "See [Page](/spaces/SPACE/pages/12345/Some-Page) for details",
			baseURL: "https://wiki.example.com",
			want:    "See [Page](https://wiki.example.com/spaces/SPACE/pages/12345/Some-Page) for details",
		},
		{
			name:    "viewpage action absolute",
			input:   "See [Page](/pages/viewpage.action?pageId=12345) for details",
			baseURL: "https://wiki.example.com",
			want:    "See [Page](https://wiki.example.com/pages/viewpage.action?pageId=12345) for details",
		},
		{
			name:    "resolve confluence pageId scheme unchanged",
			input:   "See [Page](confluence://pageId/12345) for details",
			baseURL: "https://wiki.example.com",
			want:    "See [Page](confluence://pageId/12345) for details",
		},
		{
			name:    "escaped brackets in link text preserved",
			input:   `See [\[Core\]\[SQL\] Arbitrary title](/spaces/SPACE/pages/12345/Some-Page) for details`,
			baseURL: "https://wiki.example.com",
			want:    `See [\[Core\]\[SQL\] Arbitrary title](https://wiki.example.com/spaces/SPACE/pages/12345/Some-Page) for details`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixMarkdownLinks(tt.input, tt.baseURL); got != tt.want {
				t.Fatalf("fixMarkdownLinks(%q, %q) = %q, want %q", tt.input, tt.baseURL, got, tt.want)
			}
		})
	}
}

func TestConverterConvertHTMLConfluenceAnchorHref(t *testing.T) {
	conv := NewConverter(nil)

	html := `<p>Вызвать процедуру … <a href="/pages/viewpage.action?pageId=2842940999" data-linked-resource-id="2842940999" data-linked-resource-type="page">[Core][SQL][ХП] op.program_get_by_prog_name</a></p>`

	got, err := conv.ConvertHTML(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "/pages/viewpage.action?pageId=2842940999") {
		t.Fatalf("expected view-style anchor as markdown link, got %q", got)
	}
	if !strings.Contains(got, "Вызвать процедуру … ") {
		t.Fatalf("expected space preserved before link, got %q", got)
	}
}

func TestConverterConvertHTMLConfluenceAnchorHrefSpacesPath(t *testing.T) {
	conv := NewConverter(nil)
	conv.confluenceBaseURL = "https://wiki.example.com"
	conv.plugin.SetBaseURL("https://wiki.example.com")

	html := `<p>See <a href="/spaces/XXXXX/pages/123456789/Asdfghj">Target</a> here</p>`

	got, err := conv.convertHtml(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `[Target](https://wiki.example.com/spaces/XXXXX/pages/123456789/Asdfghj)`
	if !strings.Contains(got, want) {
		t.Fatalf("expected spaces-path anchor as absolute markdown link, got %q", got)
	}
}

func TestConverterConvertHTMLConfluenceAnchorHrefWithBaseURL(t *testing.T) {
	conv := NewConverter(nil)
	conv.confluenceBaseURL = "https://wiki.example.com"
	conv.plugin.SetBaseURL("https://wiki.example.com")

	html := `<p>See <a href="/pages/viewpage.action?pageId=2842940999" data-linked-resource-id="2842940999" data-linked-resource-type="page">Target Page</a></p>`

	got, err := conv.convertHtml(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "https://wiki.example.com/pages/viewpage.action?pageId=2842940999") {
		t.Fatalf("expected absolute page link, got %q", got)
	}
}

func TestFixHTMLAnchorLinks(t *testing.T) {
	input := `See <a href="/spaces/SPACE/pages/123456789/Asdfghj">Target</a> here`
	got := fixHTMLAnchorLinks(input, "https://wiki.example.com")
	want := `[Target](https://wiki.example.com/spaces/SPACE/pages/123456789/Asdfghj)`
	if !strings.Contains(got, want) {
		t.Fatalf("fixHTMLAnchorLinks() = %q, want substring %q", got, want)
	}
}

func TestPreprocessConfluencePageAnchorsSpacesPath(t *testing.T) {
	input := `<a href="/spaces/SPACE/pages/123/Title">Link</a>`
	got := preprocessConfluencePageAnchors(input, "https://wiki.example.com")
	want := `href="https://wiki.example.com/spaces/SPACE/pages/123/Title"`
	if !strings.Contains(got, want) {
		t.Fatalf("expected absolute spaces href, got %q", got)
	}
}

func TestPreprocessConfluencePageAnchorsSingleQuotedHref(t *testing.T) {
	input := `<a href='/pages/viewpage.action?pageId=42'>Title</a>`
	got := preprocessConfluencePageAnchors(input, "https://wiki.example.com")
	if !strings.Contains(got, `href="https://wiki.example.com/pages/viewpage.action?pageId=42"`) {
		t.Fatalf("expected absolute href for single-quoted anchor, got %q", got)
	}
}

func TestPreprocessConfluencePageAnchors(t *testing.T) {
	input := `<a href="/pages/viewpage.action?pageId=42" data-linked-resource-id="42" data-linked-resource-type="page">Title</a>`
	got := preprocessConfluencePageAnchors(input, "")
	if !strings.Contains(got, `href="/pages/viewpage.action?pageId=42"`) {
		t.Fatalf("expected relative href preserved, got %q", got)
	}

	got = preprocessConfluencePageAnchors(input, "https://wiki.example.com")
	if !strings.Contains(got, `href="https://wiki.example.com/pages/viewpage.action?pageId=42"`) {
		t.Fatalf("expected absolute href, got %q", got)
	}
}

func TestFixNestedListSpacing(t *testing.T) {
	input := "\n- Item\n\n  - Nested\n\n    - Deep"
	want := "\n- Item\n  - Nested\n    - Deep"
	if got := fixNestedListSpacing(input); got != want {
		t.Fatalf("fixNestedListSpacing(%q) = %q, want %q", input, got, want)
	}
}
