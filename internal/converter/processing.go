package converter

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/javasaves/confluence-md/internal/converter/model"
	"github.com/javasaves/confluence-md/internal/converter/plugin"
	htmlnode "golang.org/x/net/html"
)

// convertHtml converts raw Confluence HTML into Markdown text.
func (c *Converter) convertHtml(html string) (string, error) {
	processedHTML := c.preprocessCDATA(html)
	processedHTML = rewriteInlineStructuredMacros(processedHTML)
	if strings.TrimSpace(c.confluenceBaseURL) != "" {
		processedHTML = preprocessConfluencePageAnchors(processedHTML, c.confluenceBaseURL)
	}

	md, err := c.mdConverter.ConvertString(processedHTML)
	if err != nil {
		fmt.Printf("Conversion error: %v\n", err)
	}

	return c.postprocessMarkdown(md), nil
}

// postprocessMarkdown normalizes whitespace and link formatting in Markdown output.
func (c *Converter) postprocessMarkdown(markdown string) string {
	markdown = regexp.MustCompile(`\n{3,}`).ReplaceAllString(markdown, "\n\n")
	markdown = fixNestedListSpacing(markdown)
	markdown = fixMarkdownLinks(markdown, c.confluenceBaseURL)
	markdown = fixHTMLAnchorLinks(markdown, c.confluenceBaseURL)

	return strings.TrimSpace(markdown)
}

// extractImageReferences finds image attachments referenced in the Confluence HTML.
func (c *Converter) extractImageReferences(html, pageID, baseURL string) []model.ImageRef {
	var imageRefs []model.ImageRef

	acImageRegex := regexp.MustCompile(`<ac:image[^>]*>[\s\S]*?</ac:image>`)
	matches := acImageRegex.FindAllString(html, -1)

	for _, imageHTML := range matches {
		fileName := plugin.ParseConfluenceImage(imageHTML)
		if fileName == "" {
			continue
		}

		encodedFilename := url.QueryEscape(fileName)
		actualURL := fmt.Sprintf("%s/download/attachments/%s/%s",
			strings.TrimSuffix(baseURL, "/"), pageID, encodedFilename)

		imageRefs = append(imageRefs, model.ImageRef{
			OriginalURL: actualURL,
			FileName:    fileName,
		})
	}

	return imageRefs
}

// fixMarkdownLinks prepends baseURL to root-relative Confluence page links without changing their paths.
func fixMarkdownLinks(markdown, baseURL string) string {
	if strings.TrimSpace(baseURL) == "" {
		return markdown
	}
	return relConfluenceURLRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		sub := relConfluenceURLRegex.FindStringSubmatch(match)
		if len(sub) < 2 || !isConfluencePageHref(sub[1]) {
			return match
		}
		return "](" + plugin.EscapeMarkdownLinkURL(plugin.PrefixBaseURL(baseURL, sub[1])) + ")"
	})
}

var (
	anchorTagRegex         = regexp.MustCompile(`(?i)<a\b([^>]*)>([\s\S]*?)</a>`)
	anchorHrefAttrRegex    = regexp.MustCompile(`(?i)\s+href=(?:"[^"]*"|'[^']*')`)
	relConfluenceURLRegex  = regexp.MustCompile(`\]\((/(?:spaces/[^/]+/pages/\d+/[^)]*|pages/viewpage\.action[^)]*|display/[^)]+))\)`)
)

func fixHTMLAnchorLinks(markdown, baseURL string) string {
	return anchorTagRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		sub := anchorTagRegex.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}

		href := plugin.ExtractHrefFromAttrs(sub[1])
		if href == "" || !isConfluencePageHref(href) {
			return match
		}

		linkURL := plugin.PrefixBaseURL(baseURL, href)
		text := plugin.HTMLFragmentText(sub[2])
		if text == "" {
			text = linkURL
		}
		return plugin.RenderMarkdownLinkString(text, linkURL)
	})
}

func isConfluencePageHref(href string) bool {
	if plugin.ExtractPageIDFromHref(href) != "" {
		return true
	}
	if strings.Contains(href, "/spaces/") && strings.Contains(href, "/pages/") {
		return true
	}
	return strings.HasPrefix(href, "/display/")
}

// preprocessConfluencePageAnchors rewrites rendered Confluence page anchors for markdown conversion.
func preprocessConfluencePageAnchors(html, baseURL string) string {
	return anchorTagRegex.ReplaceAllStringFunc(html, func(match string) string {
		sub := anchorTagRegex.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}

		attrs, inner := sub[1], sub[2]
		href := plugin.ExtractHrefFromAttrs(attrs)
		if href == "" || !isConfluencePageHref(href) {
			return match
		}

		newHref := plugin.PrefixBaseURL(baseURL, href)
		if newHref == "" {
			newHref = href
		}

		cleaned := anchorHrefAttrRegex.ReplaceAllString(attrs, "")
		return fmt.Sprintf(`<a href="%s"%s>%s</a>`, newHref, cleaned, inner)
	})
}

// fixNestedListSpacing removes extraneous blank lines in nested lists.
func fixNestedListSpacing(markdown string) string {
	listMarker := `(?:[-*+]\s|\d+\.\s)`
	pattern := regexp.MustCompile(`(\n\s*` + listMarker + `[^\n]*)\n\s*\n(\s{2,}` + listMarker + `)`)
	result := pattern.ReplaceAllString(markdown, "$1\n$2")
	if result != markdown {
		return fixNestedListSpacing(result)
	}
	return result
}

// preprocessCDATA preserves content inside CDATA nodes prior to HTML parsing.
func (c *Converter) preprocessCDATA(html string) string {
	cdataRegex := regexp.MustCompile(`<!\[CDATA\[([\s\S]*?)\]\]>`)
	return cdataRegex.ReplaceAllStringFunc(html, func(match string) string {
		if submatch := cdataRegex.FindStringSubmatch(match); len(submatch) > 1 {
			content := submatch[1]
			content = strings.ReplaceAll(content, "&", "&amp;")
			content = strings.ReplaceAll(content, "<", "&lt;")
			content = strings.ReplaceAll(content, ">", "&gt;")
			return fmt.Sprintf("<pre data-cdata='true'>%s</pre>", content)
		}
		return match
	})
}

func rewriteInlineStructuredMacros(markup string) string {
	context := &htmlnode.Node{Type: htmlnode.ElementNode, Data: "body"}
	nodes, err := htmlnode.ParseFragment(strings.NewReader(markup), context)
	if err != nil {
		return markup
	}

	for _, node := range nodes {
		rewriteInlineStructuredMacroNodes(node)
	}

	var buf strings.Builder
	for _, node := range nodes {
		_ = htmlnode.Render(&buf, node)
	}

	return buf.String()
}

func rewriteInlineStructuredMacroNodes(n *htmlnode.Node) {
	if n == nil {
		return
	}

	if n.Type == htmlnode.ElementNode &&
		n.Data == "ac:structured-macro" &&
		shouldRenderMacroInline(extractMacroNameFromNode(n)) {
		n.Data = "ac:inline-structured-macro"
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		rewriteInlineStructuredMacroNodes(child)
	}
}

func extractMacroNameFromNode(n *htmlnode.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "ac:name" {
			return strings.TrimSpace(strings.ToLower(attr.Val))
		}
	}

	return ""
}

func shouldRenderMacroInline(name string) bool {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "info", "warning", "note", "tip", "code", "mermaid-cloud", "expand", "details", "toc", "children":
		return false
	default:
		return true
	}
}
