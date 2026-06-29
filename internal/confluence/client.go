//go:generate go tool go.uber.org/mock/mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=./mock/mock_$GOFILE
package confluence

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackchuka/confluence-md/internal/confluence/model"
	"github.com/jackchuka/confluence-md/internal/version"
	"golang.org/x/net/http/httpguts"
)

type Client interface {
	GetPage(pageID string) (*model.ConfluencePage, error)
	GetChildPages(pageID string) ([]*model.ConfluencePage, error)
	DownloadAttachmentContent(attachment *model.ConfluenceAttachment) ([]byte, error)
	GetUser(accountID string) (*model.ConfluenceUser, error)
}

type AuthMode string

const (
	AuthModeBearer AuthMode = "bearer"
	AuthModeBasic  AuthMode = "basic"
)

type AuthConfig struct {
	Mode     AuthMode
	Username string
	Secret   string
}

// client represents a Confluence API client
type client struct {
	originURL  string
	basePath   string
	auth       AuthConfig
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a new Confluence API client
func NewClient(baseURL string, auth AuthConfig) Client {
	if auth.Mode != AuthModeBasic {
		auth.Mode = AuthModeBearer
	}

	originURL, basePath := splitBaseURL(baseURL)

	return &client{
		originURL: originURL,
		basePath:  basePath,
		auth:      auth,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		userAgent: fmt.Sprintf("ConfluenceMd/%s", version.Short()),
	}
}

func splitBaseURL(baseURL string) (string, string) {
	trimmed := strings.TrimSuffix(baseURL, "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return trimmed, ""
	}

	basePath := strings.TrimSuffix(parsed.Path, "/")
	if basePath == "/" {
		basePath = ""
	}

	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host), basePath
}

func (c *client) apiURL(path string) string {
	return c.originURL + c.basePath + path
}

// GetPage retrieves a Confluence page by ID
func (c *client) GetPage(pageID string) (*model.ConfluencePage, error) {
	// Build URL with expansions to get all needed data
	endpoint := fmt.Sprintf("/rest/api/content/%s", pageID)
	params := url.Values{
		"expand": []string{
			"body.storage,metadata.labels,version,space,history,children.attachment",
		},
	}

	fullURL := c.apiURL(endpoint) + "?" + params.Encode()

	resp, err := c.makeRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get page %s: %w", pageID, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp, fmt.Sprintf("get page %s", pageID))
	}

	var apiPage model.ConfluenceAPIPage
	if err := json.NewDecoder(resp.Body).Decode(&apiPage); err != nil {
		return nil, fmt.Errorf("failed to decode page response: %w", err)
	}

	// Convert API response to our model
	page := model.ConvertAPIPageToModel(&apiPage)

	return page, nil
}

const defaultChildPageLimit = 100

// GetChildPages retrieves all child pages for a given page ID
func (c *client) GetChildPages(pageID string) ([]*model.ConfluencePage, error) {
	endpoint := fmt.Sprintf("/rest/api/content/%s/child/page", pageID)
	params := url.Values{
		"expand": []string{"body.storage,metadata.labels,version,space,history"},
		"limit":  []string{strconv.Itoa(defaultChildPageLimit)},
	}

	var childPages []*model.ConfluencePage
	start := 0

	for {
		params.Set("start", strconv.Itoa(start))
		fullURL := c.apiURL(endpoint) + "?" + params.Encode()

		resp, err := c.makeRequest("GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get child pages for %s: %w", pageID, err)
		}

		if resp.StatusCode != http.StatusOK {
			err := c.handleErrorResponse(resp, fmt.Sprintf("get child pages for %s", pageID))
			_ = resp.Body.Close()
			return nil, err
		}

		var searchResult model.ConfluenceSearchResult
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode child pages response: %w", err)
		}
		_ = resp.Body.Close()

		for _, apiPage := range searchResult.Results {
			page := model.ConvertAPIPageToModel(&apiPage)
			childPages = append(childPages, page)
		}

		count := len(searchResult.Results)
		if count == 0 {
			break
		}

		limit := searchResult.Limit
		if limit <= 0 {
			limit = defaultChildPageLimit
		}

		if count < limit {
			break
		}

		start += limit
	}

	return childPages, nil
}

func (c *client) applyAuth(req *http.Request) error {
	if c.auth.Mode == AuthModeBasic {
		req.SetBasicAuth(c.auth.Username, c.auth.Secret)
		return nil
	}

	secret := strings.TrimSpace(c.auth.Secret)
	headerValue := "Bearer " + secret
	if !httpguts.ValidHeaderFieldValue(headerValue) {
		return fmt.Errorf("invalid Bearer token for Authorization header: the token contains a newline or other control character; re-save it without extra whitespace in --api-token or the OS credential store")
	}

	req.Header.Set("Authorization", headerValue)
	return nil
}

// makeRequest makes an HTTP request with authentication
func (c *client) makeRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication
	if err := c.applyAuth(req); err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// DownloadAttachmentContent downloads attachment binary content
func (c *client) DownloadAttachmentContent(attachment *model.ConfluenceAttachment) ([]byte, error) {
	if attachment == nil {
		return nil, fmt.Errorf("attachment is nil")
	}

	if attachment.DownloadLink == "" {
		return nil, fmt.Errorf("attachment %s has no download link", attachment.Title)
	}

	downloadURL, err := c.normalizeDownloadLink(attachment.DownloadLink)
	if err != nil {
		return nil, err
	}

	urls := []string{downloadURL}
	// Some Confluence Cloud sites reject API-token auth on the legacy
	// /download/ media path (responding 401 with www-authenticate: OAuth).
	// The v1 REST attachment endpoint honors token auth, so try it as a fallback.
	if fallbackURL, ok := c.attachmentRESTDownloadURL(attachment); ok {
		urls = append(urls, fallbackURL)
	}

	var lastResp *http.Response
	for _, u := range urls {
		resp, err := c.fetchBinary(u)
		if err != nil {
			return nil, fmt.Errorf("failed to download attachment %s: %w", attachment.Title, err)
		}

		if resp.StatusCode == http.StatusOK {
			defer func() {
				_ = resp.Body.Close()
			}()
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read attachment content: %w", err)
			}
			return data, nil
		}

		if lastResp != nil {
			_ = lastResp.Body.Close()
		}
		lastResp = resp
	}

	defer func() {
		_ = lastResp.Body.Close()
	}()
	return nil, c.handleErrorResponse(lastResp, fmt.Sprintf("download attachment %s", attachment.Title))
}

// fetchBinary issues an authenticated GET for raw attachment bytes.
func (c *client) fetchBinary(downloadURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.applyAuth(req); err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", c.userAgent)

	return c.httpClient.Do(req)
}

// attachmentRESTDownloadURL builds the v1 REST download URL for an attachment,
// which accepts token auth where the legacy download path may not.
func (c *client) attachmentRESTDownloadURL(attachment *model.ConfluenceAttachment) (string, bool) {
	if attachment.ID == "" {
		return "", false
	}

	pageID, ok := pageIDFromDownloadLink(attachment.DownloadLink)
	if !ok {
		return "", false
	}

	return c.apiURL(fmt.Sprintf("/rest/api/content/%s/child/attachment/%s/download",
		pageID, attachment.ID)), true
}

// pageIDFromDownloadLink extracts the parent page ID from a download link of the
// form /download/attachments/{pageID}/{filename}?...
func pageIDFromDownloadLink(link string) (string, bool) {
	_, rest, found := strings.Cut(link, "/attachments/")
	if !found {
		return "", false
	}

	pageID, _, found := strings.Cut(rest, "/")
	if !found || pageID == "" {
		return "", false
	}

	return pageID, true
}

func (c *client) normalizeDownloadLink(link string) (string, error) {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link, nil
	}

	if !strings.HasPrefix(link, "/") {
		link = "/" + link
	}

	if c.basePath != "" && strings.HasPrefix(link, "/download/") {
		link = c.basePath + link
	}

	if strings.Contains(link, " ") {
		link = strings.ReplaceAll(link, " ", "%20")
	}

	full := c.originURL + link
	parsed, err := url.Parse(full)
	if err != nil {
		return "", fmt.Errorf("invalid attachment url %s: %w", full, err)
	}
	return parsed.String(), nil
}

// GetUser retrieves user information by account ID
func (c *client) GetUser(accountID string) (*model.ConfluenceUser, error) {
	endpoint := fmt.Sprintf("/rest/api/user?accountId=%s", url.QueryEscape(accountID))
	fullURL := c.apiURL(endpoint)

	resp, err := c.makeRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", accountID, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp, fmt.Sprintf("get user %s", accountID))
	}

	var user model.ConfluenceUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}

// handleErrorResponse handles error responses from the API
func (c *client) handleErrorResponse(resp *http.Response, operation string) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to %s: HTTP %d", operation, resp.StatusCode)
	}

	// Try to parse error response
	var errorResp model.ConfluenceErrorResponse
	if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
		return fmt.Errorf("failed to %s: %s", operation, errorResp.Message)
	}

	// Fallback to HTTP status
	return fmt.Errorf("failed to %s: HTTP %d - %s", operation, resp.StatusCode, string(bodyBytes))
}
