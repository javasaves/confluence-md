package credentials

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

type staticSecretStore struct {
	secret string
	err    error
}

func (s staticSecretStore) Get(service, account string) (string, error) {
	return s.secret, s.err
}

func TestBuildReferenceNormalizesBaseURL(t *testing.T) {
	tests := []struct {
		name           string
		mode           Mode
		baseURL        string
		email          string
		wantBaseURL    string
		wantAccountKey string
	}{
		{
			name:           "root context lowercases scheme and host",
			mode:           ModeBearer,
			baseURL:        "HTTPS://Wiki.Company.Local:443/?debug=1#fragment",
			wantBaseURL:    "https://wiki.company.local",
			wantAccountKey: "bearer|https://wiki.company.local",
		},
		{
			name:           "wiki context removes trailing slash and default https port",
			mode:           ModeBearer,
			baseURL:        "https://Example.Atlassian.Net:443/wiki/?q=1#frag",
			wantBaseURL:    "https://example.atlassian.net/wiki",
			wantAccountKey: "bearer|https://example.atlassian.net/wiki",
		},
		{
			name:           "custom context keeps non default port",
			mode:           ModeBearer,
			baseURL:        "https://Wiki.Company.Local:8443/confluence/?q=1",
			wantBaseURL:    "https://wiki.company.local:8443/confluence",
			wantAccountKey: "bearer|https://wiki.company.local:8443/confluence",
		},
		{
			name:           "basic lookup includes email selector",
			mode:           ModeBasic,
			baseURL:        "http://wiki.company.local:80/wiki/",
			email:          "john.doe@example.com",
			wantBaseURL:    "http://wiki.company.local/wiki",
			wantAccountKey: "basic|http://wiki.company.local/wiki|john.doe@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := BuildReference(tt.mode, tt.baseURL, tt.email)
			if err != nil {
				t.Fatalf("BuildReference() error = %v", err)
			}

			if ref.ServiceID != ServiceID {
				t.Fatalf("unexpected service ID %q, want %q", ref.ServiceID, ServiceID)
			}

			if ref.BaseURL != tt.wantBaseURL {
				t.Fatalf("unexpected normalized base URL %q, want %q", ref.BaseURL, tt.wantBaseURL)
			}

			if ref.AccountKey != tt.wantAccountKey {
				t.Fatalf("unexpected account key %q, want %q", ref.AccountKey, tt.wantAccountKey)
			}
		})
	}
}

func TestBuildReferenceBasicRequiresEmail(t *testing.T) {
	_, err := BuildReference(ModeBasic, "https://example.atlassian.net/wiki", "")
	if err == nil {
		t.Fatal("expected missing email to fail")
	}

	if got, want := err.Error(), "email is required for basic auth store lookups"; got != want {
		t.Fatalf("unexpected error %q, want %q", got, want)
	}
}

func TestLookupSecretDecodesUTF16LESecretsFromStore(t *testing.T) {
	const wantSecret = "Njk+/token"

	secret, _, err := LookupSecret(
		staticSecretStore{secret: encodeUTF16LEString(wantSecret, true)},
		ModeBearer,
		"https://example.atlassian.net/wiki",
		"",
	)
	if err != nil {
		t.Fatalf("LookupSecret() error = %v", err)
	}

	if secret != wantSecret {
		t.Fatalf("unexpected decoded secret %q, want %q", secret, wantSecret)
	}
}

func TestNormalizeStoredSecretLeavesPlainSecretsUntouched(t *testing.T) {
	const secret = "plain-token+/value"

	if got := normalizeStoredSecret(secret); got != secret {
		t.Fatalf("unexpected normalized secret %q, want %q", got, secret)
	}
}

func encodeUTF16LEString(value string, addTerminator bool) string {
	units := utf16.Encode([]rune(value))
	if addTerminator {
		units = append(units, 0)
	}

	buf := make([]byte, 0, len(units)*2)
	for _, unit := range units {
		var pair [2]byte
		binary.LittleEndian.PutUint16(pair[:], unit)
		buf = append(buf, pair[:]...)
	}

	return string(buf)
}
