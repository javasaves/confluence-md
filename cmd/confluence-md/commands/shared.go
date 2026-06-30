package commands

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"github.com/javasaves/confluence-md/internal/confluence"
	confluenceModel "github.com/javasaves/confluence-md/internal/confluence/model"
	"github.com/javasaves/confluence-md/internal/converter"
)

// sanitizeFileName uses the mature gosimple/slug library for robust filename sanitization
func sanitizeFileName(name string) string {
	if name == "" {
		return "untitled"
	}

	sanitized := slug.MakeLang(name, "en")

	if sanitized == "" {
		return name
	}

	return sanitized
}

func buildOutputNamer(template string) (converter.OutputNamer, error) {
	if strings.TrimSpace(template) == "" {
		return nil, nil
	}

	namer, err := converter.NewTemplateOutputNamer(template)
	if err != nil {
		return nil, err
	}

	return namer, nil
}

// PageConversionResult represents the result of converting a single page
type PageConversionResult struct {
	OutputPath  string
	PageID      string
	Title       string
	ImagesCount int
	Success     bool
	Error       error
}

// convertSinglePage handles the full conversion pipeline for a single page
func convertSinglePage(client confluence.Client, page *confluenceModel.ConfluencePage, baseURL string, opts PageOptions) *PageConversionResult {
	return convertSinglePageWithPath(client, page, baseURL, "", opts)
}

// convertSinglePageWithPath handles conversion with a custom output path (for tree structure)
func convertSinglePageWithPath(client confluence.Client, page *confluenceModel.ConfluencePage, baseURL, outputPath string, opts PageOptions) *PageConversionResult {
	result := &PageConversionResult{
		PageID: page.ID,
		Title:  page.Title,
	}

	if outputPath == "" {
		fileName, err := converter.GenerateFileName(page, opts.OutputNamer)
		if err != nil {
			result.Error = fmt.Errorf("failed to generate output filename: %w", err)
			return result
		}
		outputPath = filepath.Join(opts.OutputDir, fileName)
	}
	result.OutputPath = outputPath

	// Create converter and convert page
	var options []converter.Option
	if opts.DownloadImages {
		options = append(options, converter.WithDownloadAttachments(opts.ImageFolder))
	}
	conv := converter.NewConverter(client, options...)
	doc, err := conv.ConvertPage(page, baseURL, filepath.Dir(outputPath), opts.SourcePageURL)
	if err != nil {
		result.Error = fmt.Errorf("failed to convert page: %w", err)
		return result
	}
	result.ImagesCount = len(doc.Images)

	if err := converter.SaveMarkdownDocument(doc, outputPath, opts.IncludeMetadata); err != nil {
		result.Error = fmt.Errorf("failed to save document: %w", err)
		return result
	}

	result.Success = true
	return result
}

// printConversionResult prints the result of a page conversion in a consistent format
func printConversionResult(result *PageConversionResult) {
	if result.Success {
		fmt.Printf("✅ Successfully converted page: %s\n", result.OutputPath)
		fmt.Printf("   Page ID: %s\n", result.PageID)
		fmt.Printf("   Title: %s\n", result.Title)
		if result.ImagesCount > 0 {
			fmt.Printf("   📥 Images downloaded: %d\n", result.ImagesCount)
		}
	} else {
		fmt.Printf("❌ Failed to convert page: %s\n", result.Title)
		if result.Error != nil {
			fmt.Printf("   Error: %v\n", result.Error)
		}
	}
	fmt.Println()
}

const (
	errCouldNotExtractPageID        = "could not extract page ID from URL"
	errCouldNotExtractPageIDOrTitle = "could not extract page ID or title from URL"
)

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func basePathBeforeAnchor(pathParts []string, endIndex int) string {
	for _, anchor := range []string{"spaces", "display", "pages"} {
		for i := 0; i < endIndex; i++ {
			if pathParts[i] == anchor {
				if i > 0 {
					return "/" + strings.Join(pathParts[:i], "/")
				}
				return ""
			}
		}
	}
	return ""
}

func extractSpaceKeyFromPath(pathParts []string, endIndex int) string {
	for i := 0; i < endIndex; i++ {
		if pathParts[i] == "spaces" && i+1 < endIndex {
			return pathParts[i+1]
		}
	}
	return ""
}

func decodeDisplayTitle(raw string) (string, error) {
	decoded, err := url.PathUnescape(strings.ReplaceAll(raw, "+", " "))
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func ensurePageID(client confluence.Client, info confluenceModel.PageURLInfo) (confluenceModel.PageURLInfo, error) {
	if info.PageID != "" {
		return info, nil
	}
	id, err := client.FindPageIDByTitle(info.SpaceKey, info.Title)
	if err != nil {
		return info, err
	}
	info.PageID = id
	return info, nil
}

func urlToPageInfo(pageURL string) (confluenceModel.PageURLInfo, error) {
	if pageURL == "" {
		return confluenceModel.PageURLInfo{}, fmt.Errorf("URL is empty")
	}

	u, err := url.Parse(pageURL)
	if err != nil {
		return confluenceModel.PageURLInfo{}, fmt.Errorf("invalid URL: %w", err)
	}

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	var pageID string
	var spaceKey string
	var title string

	if len(pathParts) > 0 && pathParts[len(pathParts)-1] == "viewpage.action" {
		viewpageIndex := len(pathParts) - 1
		pageID = u.Query().Get("pageId")
		if pageID != "" && !isAllDigits(pageID) {
			return confluenceModel.PageURLInfo{}, fmt.Errorf(errCouldNotExtractPageID)
		}
		title = u.Query().Get("title")
		spaceKey = extractSpaceKeyFromPath(pathParts, viewpageIndex)
		if querySpaceKey := u.Query().Get("spaceKey"); querySpaceKey != "" {
			spaceKey = querySpaceKey
		}
		basePath := basePathBeforeAnchor(pathParts, viewpageIndex)
		baseURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, basePath)
		if pageID == "" && title == "" {
			return confluenceModel.PageURLInfo{}, fmt.Errorf(errCouldNotExtractPageIDOrTitle)
		}
		return confluenceModel.PageURLInfo{
			BaseURL:   baseURL,
			SourceURL: pageURL,
			PageID:    pageID,
			SpaceKey:  spaceKey,
			Title:     title,
		}, nil
	}

	for i, part := range pathParts {
		if part == "display" && i+1 < len(pathParts) {
			spaceKey = pathParts[i+1]
			rawTitle := pathParts[len(pathParts)-1]
			decoded, err := decodeDisplayTitle(rawTitle)
			if err != nil {
				return confluenceModel.PageURLInfo{}, fmt.Errorf("invalid URL: %w", err)
			}
			title = decoded
			basePath := ""
			if i > 0 {
				basePath = "/" + strings.Join(pathParts[:i], "/")
			}
			if queryPageID := u.Query().Get("pageId"); queryPageID != "" {
				if !isAllDigits(queryPageID) {
					return confluenceModel.PageURLInfo{}, fmt.Errorf(errCouldNotExtractPageID)
				}
				pageID = queryPageID
			}
			baseURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, basePath)
			if pageID == "" && title == "" {
				return confluenceModel.PageURLInfo{}, fmt.Errorf(errCouldNotExtractPageIDOrTitle)
			}
			return confluenceModel.PageURLInfo{
				BaseURL:   baseURL,
				SourceURL: pageURL,
				PageID:    pageID,
				SpaceKey:  spaceKey,
				Title:     title,
			}, nil
		}
	}

	basePath := ""
	spaceKeyIndex := -1
	for i, part := range pathParts {
		if part == "spaces" && i+1 < len(pathParts) {
			spaceKey = pathParts[i+1]
			spaceKeyIndex = i
		}
		if part == "pages" && i+1 < len(pathParts) {
			pageID = pathParts[i+1]
		}
		if i == len(pathParts)-1 {
			title = part
		}
	}

	if queryPageID := u.Query().Get("pageId"); queryPageID != "" {
		if !isAllDigits(queryPageID) {
			return confluenceModel.PageURLInfo{}, fmt.Errorf(errCouldNotExtractPageID)
		}
		pageID = queryPageID
	}

	if spaceKeyIndex > 0 {
		basePath = "/" + strings.Join(pathParts[:spaceKeyIndex], "/")
	}

	baseURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, basePath)

	if pageID == "" || !isAllDigits(pageID) {
		return confluenceModel.PageURLInfo{}, fmt.Errorf(errCouldNotExtractPageID)
	}

	return confluenceModel.PageURLInfo{
		BaseURL:   baseURL,
		SourceURL: pageURL,
		PageID:    pageID,
		SpaceKey:  spaceKey,
		Title:     title,
	}, nil
}
