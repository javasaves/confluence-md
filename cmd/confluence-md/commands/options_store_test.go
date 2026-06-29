package commands

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/jackchuka/confluence-md/internal/confluence"
	"github.com/jackchuka/confluence-md/internal/credentials"
	"github.com/zalando/go-keyring"
)

type fakeSecretStore struct {
	secrets      map[string]string
	err          error
	errByAccount map[string]error
}

func (s fakeSecretStore) Get(service, account string) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	if err, ok := s.errByAccount[account]; ok {
		return "", err
	}

	secret, ok := s.secrets[storeSecretKey(service, account)]
	if !ok {
		return "", keyring.ErrNotFound
	}

	return secret, nil
}

func storeSecretKey(service, account string) string {
	return service + "\n" + account
}

func encodeUTF16LESecret(secret string) string {
	units := append(utf16.Encode([]rune(secret)), 0)
	buf := make([]byte, 0, len(units)*2)
	for _, unit := range units {
		var pair [2]byte
		binary.LittleEndian.PutUint16(pair[:], unit)
		buf = append(buf, pair[:]...)
	}

	return string(buf)
}

func TestAuthOptionsResolveManualBearer(t *testing.T) {
	opts := authOptions{APIKey: "pat-token"}

	auth, err := opts.Resolve("https://example.atlassian.net/wiki")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.Mode != confluence.AuthModeBearer {
		t.Fatalf("unexpected auth mode %q, want %q", auth.Mode, confluence.AuthModeBearer)
	}

	if auth.Secret != "pat-token" {
		t.Fatalf("unexpected secret %q", auth.Secret)
	}

	if auth.Username != "" {
		t.Fatalf("unexpected username %q for bearer auth", auth.Username)
	}
}

func TestAuthOptionsResolveManualBasic(t *testing.T) {
	opts := authOptions{
		APIKey:    "secret",
		Email:     " john.doe@example.com ",
		BasicAuth: true,
	}

	auth, err := opts.Resolve("https://example.atlassian.net/wiki")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.Mode != confluence.AuthModeBasic {
		t.Fatalf("unexpected auth mode %q, want %q", auth.Mode, confluence.AuthModeBasic)
	}

	if auth.Username != "john.doe@example.com" {
		t.Fatalf("unexpected username %q", auth.Username)
	}

	if auth.Secret != "secret" {
		t.Fatalf("unexpected secret %q", auth.Secret)
	}
}

func TestAuthOptionsResolveStoredBearer(t *testing.T) {
	baseURL := "https://Example.Atlassian.Net:443/wiki/?debug=1#fragment"
	ref, err := credentials.BuildReference(credentials.ModeBearer, baseURL, "")
	if err != nil {
		t.Fatalf("BuildReference() error = %v", err)
	}

	opts := authOptions{
		BearerAuthStore: true,
		secretStore: fakeSecretStore{
			secrets: map[string]string{
				storeSecretKey(ref.ServiceID, ref.AccountKey): "stored-pat",
			},
		},
	}

	auth, err := opts.Resolve(baseURL)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.Mode != confluence.AuthModeBearer {
		t.Fatalf("unexpected auth mode %q, want %q", auth.Mode, confluence.AuthModeBearer)
	}

	if auth.Secret != "stored-pat" {
		t.Fatalf("unexpected secret %q", auth.Secret)
	}
}

func TestAuthOptionsResolveStoredBearerDecodesUTF16LESecret(t *testing.T) {
	baseURL := "https://Example.Atlassian.Net:443/wiki/?debug=1#fragment"
	ref, err := credentials.BuildReference(credentials.ModeBearer, baseURL, "")
	if err != nil {
		t.Fatalf("BuildReference() error = %v", err)
	}

	opts := authOptions{
		BearerAuthStore: true,
		secretStore: fakeSecretStore{
			secrets: map[string]string{
				storeSecretKey(ref.ServiceID, ref.AccountKey): encodeUTF16LESecret("Njk+/token"),
			},
		},
	}

	auth, err := opts.Resolve(baseURL)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.Secret != "Njk+/token" {
		t.Fatalf("unexpected decoded secret %q", auth.Secret)
	}
}

func TestAuthOptionsResolveStoredBasic(t *testing.T) {
	baseURL := "https://example.atlassian.net/wiki/"
	email := " john.doe@example.com "
	ref, err := credentials.BuildReference(credentials.ModeBasic, baseURL, email)
	if err != nil {
		t.Fatalf("BuildReference() error = %v", err)
	}

	opts := authOptions{
		Email:          email,
		BasicAuthStore: true,
		secretStore: fakeSecretStore{
			secrets: map[string]string{
				storeSecretKey(ref.ServiceID, ref.AccountKey): "stored-basic-secret",
			},
		},
	}

	auth, err := opts.Resolve(baseURL)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if auth.Mode != confluence.AuthModeBasic {
		t.Fatalf("unexpected auth mode %q, want %q", auth.Mode, confluence.AuthModeBasic)
	}

	if auth.Username != "john.doe@example.com" {
		t.Fatalf("unexpected username %q, want %q", auth.Username, "john.doe@example.com")
	}

	if auth.Secret != "stored-basic-secret" {
		t.Fatalf("unexpected secret %q", auth.Secret)
	}
}

func TestAuthOptionsValidateRejectsConflictingModes(t *testing.T) {
	tests := []struct {
		name    string
		opts    authOptions
		wantErr string
	}{
		{
			name: "both store flags",
			opts: authOptions{
				BearerAuthStore: true,
				BasicAuthStore:  true,
			},
			wantErr: "flags --bearer-auth-store and --basic-auth-store are mutually exclusive",
		},
		{
			name: "api token with bearer store",
			opts: authOptions{
				APIKey:          "pat",
				BearerAuthStore: true,
			},
			wantErr: "flag --api-token cannot be used with --bearer-auth-store",
		},
		{
			name: "api token with basic store",
			opts: authOptions{
				APIKey:         "secret",
				BasicAuthStore: true,
			},
			wantErr: "flag --api-token cannot be used with --basic-auth-store",
		},
		{
			name: "basic auth with basic store",
			opts: authOptions{
				BasicAuth:      true,
				BasicAuthStore: true,
			},
			wantErr: "flag --basic-auth cannot be used with --basic-auth-store",
		},
		{
			name: "basic auth with bearer store",
			opts: authOptions{
				BasicAuth:       true,
				BearerAuthStore: true,
			},
			wantErr: "flag --basic-auth cannot be used with --bearer-auth-store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}

			if got := err.Error(); got != tt.wantErr {
				t.Fatalf("unexpected error %q, want %q", got, tt.wantErr)
			}
		})
	}
}

func TestAuthOptionsResolveStoredBasicRequiresEmail(t *testing.T) {
	opts := authOptions{BasicAuthStore: true}

	_, err := opts.Resolve("https://example.atlassian.net/wiki")
	if err == nil {
		t.Fatal("expected missing email to fail")
	}

	if got, want := err.Error(), "missing required flag: --email when --basic-auth-store is set"; got != want {
		t.Fatalf("unexpected error %q, want %q", got, want)
	}
}

func TestAuthOptionsResolveStoredNotFoundShowsServiceAndAccount(t *testing.T) {
	baseURL := "https://Example.Atlassian.Net:443/wiki/?debug=1#fragment"
	ref, err := credentials.BuildReference(credentials.ModeBearer, baseURL, "")
	if err != nil {
		t.Fatalf("BuildReference() error = %v", err)
	}

	opts := authOptions{
		BearerAuthStore: true,
		secretStore:     fakeSecretStore{},
	}

	_, err = opts.Resolve(baseURL)
	if err == nil {
		t.Fatal("expected missing stored secret to fail")
	}

	for _, fragment := range []string{
		"no stored secret found in the OS credential store",
		ref.ServiceID,
		ref.AccountKey,
		ref.WindowsTargetName(),
		ref.WindowsUserName(),
		"Target/Address",
		"UserName",
	} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected error %q to contain %q", err.Error(), fragment)
		}
	}
}

func TestAuthOptionsResolveStoredUnavailableErrors(t *testing.T) {
	baseURL := "https://example.atlassian.net/wiki"
	tests := []struct {
		name  string
		store fakeSecretStore
		cause string
	}{
		{
			name:  "locked store",
			store: fakeSecretStore{err: errors.New("keychain is locked")},
			cause: "keychain is locked",
		},
		{
			name:  "not configured store",
			store: fakeSecretStore{err: errors.New("secret service is not configured")},
			cause: "secret service is not configured",
		},
		{
			name:  "unavailable store",
			store: fakeSecretStore{err: errors.New("credential manager is unavailable")},
			cause: "credential manager is unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := authOptions{
				BearerAuthStore: true,
				secretStore:     tt.store,
			}

			_, err := opts.Resolve(baseURL)
			if err == nil {
				t.Fatal("expected store access error")
			}

			if !strings.Contains(err.Error(), "failed to access the OS credential store") {
				t.Fatalf("expected credential store access error, got %q", err.Error())
			}

			if !strings.Contains(err.Error(), tt.cause) {
				t.Fatalf("expected error %q to contain cause %q", err.Error(), tt.cause)
			}

			for _, fragment := range []string{
				"Target/Address",
				"UserName",
				credentials.ServiceID,
			} {
				if !strings.Contains(err.Error(), fragment) {
					t.Fatalf("expected error %q to contain %q", err.Error(), fragment)
				}
			}
		})
	}
}

func TestAuthOptionsResolveStoredSecretsUseNormalizedBaseURLScope(t *testing.T) {
	const normalizedBaseURL = "https://example.atlassian.net/wiki"

	ref, err := credentials.BuildReference(credentials.ModeBearer, normalizedBaseURL, "")
	if err != nil {
		t.Fatalf("BuildReference() error = %v", err)
	}

	opts := authOptions{
		BearerAuthStore: true,
		secretStore: fakeSecretStore{
			secrets: map[string]string{
				storeSecretKey(ref.ServiceID, ref.AccountKey): "stored-pat",
			},
		},
	}

	successCases := []string{
		"https://example.atlassian.net/wiki",
		"https://Example.Atlassian.Net:443/wiki/",
		"https://example.atlassian.net/wiki?foo=bar#frag",
	}

	for _, baseURL := range successCases {
		auth, err := opts.Resolve(baseURL)
		if err != nil {
			t.Fatalf("Resolve(%q) error = %v", baseURL, err)
		}

		if auth.Secret != "stored-pat" {
			t.Fatalf("Resolve(%q) secret = %q, want %q", baseURL, auth.Secret, "stored-pat")
		}
	}

	_, err = opts.Resolve("https://example.atlassian.net/confluence")
	if err == nil {
		t.Fatal("expected different context path to miss the stored secret")
	}

	if !strings.Contains(err.Error(), "no stored secret found") {
		t.Fatalf("expected not found error for different context path, got %q", err.Error())
	}
}
