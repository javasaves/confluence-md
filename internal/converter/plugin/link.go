package plugin

import (
	"fmt"
	htmlstd "html"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func findChildElement(n *html.Node, tag string) *html.Node {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			return child
		}
	}
	return nil
}

func findDescendantElement(n *html.Node, tag string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findDescendantElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractContentID(n *html.Node) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == "ri:content-id" {
			return attr.Val
		}
	}
	return ""
}

// extractPageID reads ri:content-id when present on ri:page (non-standard extension).
func extractPageID(riPage *html.Node) string {
	return extractContentID(riPage)
}

func extractSpaceKey(riPage *html.Node) string {
	if riPage == nil {
		return ""
	}
	for _, attr := range riPage.Attr {
		if attr.Key == "ri:space-key" {
			return attr.Val
		}
	}
	return ""
}

func extractContentTitle(riPage *html.Node) string {
	if riPage == nil {
		return ""
	}
	for _, attr := range riPage.Attr {
		if attr.Key == "ri:content-title" {
			return attr.Val
		}
	}
	return ""
}

func extractURLValue(riURL *html.Node) string {
	if riURL == nil {
		return ""
	}
	for _, attr := range riURL.Attr {
		if attr.Key == "ri:value" {
			return attr.Val
		}
	}
	return ""
}

func extractConfluenceLinkText(n *html.Node) string {
	linkBody := findDescendantElement(n, "ac:plain-text-link-body")
	if linkBody != nil {
		if text := extractPlainTextLinkBody(linkBody); text != "" {
			return text
		}
	}

	riPage := findDescendantElement(n, "ri:page")
	return extractContentTitle(riPage)
}

// BuildRelativePageLinkURL returns a root-relative Confluence page URL.
func BuildRelativePageLinkURL(spaceKey, pageID, title string) string {
	if pageID == "" {
		return ""
	}
	if spaceKey != "" && title != "" {
		return fmt.Sprintf("/spaces/%s/pages/%s/%s", spaceKey, pageID, url.PathEscape(title))
	}
	return fmt.Sprintf("/pages/viewpage.action?pageId=%s", pageID)
}

// BuildRelativePageTitleURL returns a root-relative URL for a page referenced by space + title.
func BuildRelativePageTitleURL(spaceKey, title string) string {
	if title == "" {
		return ""
	}
	if spaceKey != "" && spaceKey != "_" {
		titleSegment := strings.ReplaceAll(url.PathEscape(title), "%20", "+")
		return fmt.Sprintf("/display/%s/%s", url.PathEscape(spaceKey), titleSegment)
	}
	return fmt.Sprintf("/pages/viewpage.action?title=%s", url.QueryEscape(title))
}

// BuildConfluencePageLinkURL returns an absolute Confluence page URL (e.g. for frontmatter).
func BuildConfluencePageLinkURL(baseURL, spaceKey, pageID, title string) string {
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" || pageID == "" {
		return ""
	}
	if spaceKey != "" && title != "" {
		return fmt.Sprintf("%s/spaces/%s/pages/%s/%s", base, spaceKey, pageID, url.PathEscape(title))
	}
	return fmt.Sprintf("%s/pages/viewpage.action?pageId=%s", base, pageID)
}

// BuildConfluencePageTitleURL returns a browser-openable URL for a page referenced by space + title.
func BuildConfluencePageTitleURL(baseURL, spaceKey, title string) string {
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" || title == "" {
		return ""
	}
	if spaceKey == "" || spaceKey == "_" {
		return fmt.Sprintf("%s/pages/viewpage.action?title=%s", base, url.QueryEscape(title))
	}
	titleSegment := strings.ReplaceAll(url.PathEscape(title), "%20", "+")
	return fmt.Sprintf("%s/display/%s/%s", base, url.PathEscape(spaceKey), titleSegment)
}

func RenderMarkdownLinkString(text, linkURL string) string {
	return renderMarkdownLinkString(text, linkURL)
}

func renderMarkdownLinkString(text, linkURL string) string {
	var b strings.Builder
	renderMarkdownLink(&b, text, linkURL)
	return b.String()
}

func extractPlainTextLinkBody(linkBody *html.Node) string {
	for child := linkBody.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "pre" {
			for _, attr := range child.Attr {
				if attr.Key == "data-cdata" && attr.Val == "true" {
					return unescapePreCDATAContent(child)
				}
			}
		}
	}

	return extractLinkBodyRawText(linkBody)
}

func unescapePreCDATAContent(pre *html.Node) string {
	content := collectTextContent(pre)
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&amp;", "&")
	return strings.TrimSpace(content)
}

func extractLinkBodyRawText(linkBody *html.Node) string {
	var b strings.Builder
	for child := linkBody.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case html.TextNode:
			b.WriteString(child.Data)
		case html.CommentNode:
			text := child.Data
			text = strings.TrimPrefix(text, "[CDATA[")
			text = strings.TrimSuffix(text, "]]")
			b.WriteString(text)
		case html.ElementNode:
			if child.Data == "pre" {
				b.WriteString(collectTextContent(child))
			}
		}
	}
	return strings.TrimSpace(htmlstd.UnescapeString(b.String()))
}

func collectTextContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(collectTextContent(child))
	}
	return b.String()
}

func escapeMarkdownLinkText(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	for _, r := range s {
		switch r {
		case '\\', '[', ']', '*', '_', '`':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// EscapeMarkdownLinkURL escapes characters that break markdown link destinations.
func EscapeMarkdownLinkURL(s string) string {
	return escapeMarkdownLinkURL(s)
}

func escapeMarkdownLinkURL(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/8)
	for _, r := range s {
		switch r {
		case '\\', '(', ')':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func renderMarkdownLink(w io.Writer, text, linkURL string) {
	_, _ = fmt.Fprintf(w, "[%s](%s)", escapeMarkdownLinkText(text), escapeMarkdownLinkURL(linkURL))
}

var (
	pageIDFromHrefRegex  = regexp.MustCompile(`(?i)[?&]pageId=(\d+)`)
	spacesPageIDRegex    = regexp.MustCompile(`(?i)/spaces/[^/]+/pages/(\d+)`)
	hrefAttrValueRegex   = regexp.MustCompile(`(?i)\shref=(?:"([^"]*)"|'([^']*)')`)
	stripHTMLTagsRegex   = regexp.MustCompile(`<[^>]*>`)
)

// ExtractPageIDFromHref reads a Confluence page ID from common href formats.
func ExtractPageIDFromHref(href string) string {
	if m := pageIDFromHrefRegex.FindStringSubmatch(href); len(m) > 1 {
		return m[1]
	}
	if m := spacesPageIDRegex.FindStringSubmatch(href); len(m) > 1 {
		return m[1]
	}
	return ""
}

// PrefixBaseURL prepends baseURL to a root-relative href without changing the path.
func PrefixBaseURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" || !strings.HasPrefix(href, "/") {
		return href
	}
	return base + href
}

// ResolveConfluencePageHref returns href unchanged except for prepending baseURL to root-relative paths.
func ResolveConfluencePageHref(href, baseURL string) string {
	return PrefixBaseURL(baseURL, href)
}

func extractHrefAttr(n *html.Node) string {
	if n == nil {
		return ""
	}
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			return attr.Val
		}
	}
	return ""
}

func ExtractHrefFromAttrs(attrs string) string {
	return extractHrefFromAttrs(attrs)
}

func extractHrefFromAttrs(attrs string) string {
	m := hrefAttrValueRegex.FindStringSubmatch(attrs)
	if len(m) < 2 {
		return ""
	}
	if m[1] != "" {
		return m[1]
	}
	return m[2]
}

// HTMLFragmentText returns visible text from a small HTML fragment.
func HTMLFragmentText(fragment string) string {
	return strings.TrimSpace(htmlstd.UnescapeString(stripHTMLTags(fragment)))
}

func stripHTMLTags(s string) string {
	return stripHTMLTagsRegex.ReplaceAllString(s, "")
}
