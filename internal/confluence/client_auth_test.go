package confluence

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"
)

func TestApplyAuthUsesBearerByDefault(t *testing.T) {
	c := &client{
		auth: AuthConfig{
			Mode:   AuthModeBearer,
			Secret: "pat-token",
		},
	}

	req := httptest.NewRequest("GET", "https://example.test/wiki/rest/api/content/1", nil)
	c.applyAuth(req)

	if got := req.Header.Get("Authorization"); got != "Bearer pat-token" {
		t.Fatalf("expected bearer auth header, got %q", got)
	}
}

func TestApplyAuthUsesBasicWhenRequested(t *testing.T) {
	c := &client{
		auth: AuthConfig{
			Mode:     AuthModeBasic,
			Username: "alice",
			Secret:   "secret",
		},
	}

	req := httptest.NewRequest("GET", "https://example.test/wiki/rest/api/content/1", nil)
	c.applyAuth(req)

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("expected basic auth header %q, got %q", want, got)
	}
}

func TestNewClientDefaultsToBearerAuth(t *testing.T) {
	rawClient := NewClient("https://example.test", AuthConfig{Secret: "pat-token"})
	c, ok := rawClient.(*client)
	if !ok {
		t.Fatalf("expected *client, got %T", rawClient)
	}

	if c.auth.Mode != AuthModeBearer {
		t.Fatalf("expected default auth mode %q, got %q", AuthModeBearer, c.auth.Mode)
	}
}
