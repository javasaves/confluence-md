package credentials

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/zalando/go-keyring"
)

const ServiceID = "github.com/javasaves/confluence-md/auth/v1"

type Mode string

const (
	ModeBearer Mode = "bearer"
	ModeBasic  Mode = "basic"
)

var (
	ErrSecretNotFound         = errors.New("secret not found")
	ErrSecretStoreUnavailable = errors.New("secret store unavailable")
)

type Store interface {
	Get(service, account string) (string, error)
}

type Reference struct {
	ServiceID  string
	AccountKey string
	BaseURL    string
}

func (r Reference) WindowsTargetName() string {
	return r.ServiceID + ":" + r.AccountKey
}

func (r Reference) WindowsUserName() string {
	return r.AccountKey
}

type systemStore struct{}

type StoreAccessError struct {
	Cause error
}

func DefaultStore() Store {
	return systemStore{}
}

func (systemStore) Get(service, account string) (string, error) {
	return keyring.Get(service, account)
}

func (e *StoreAccessError) Error() string {
	return e.Cause.Error()
}

func (e *StoreAccessError) Unwrap() error {
	return ErrSecretStoreUnavailable
}

func NormalizeBaseURL(rawBaseURL string) (string, error) {
	trimmed := strings.TrimSpace(rawBaseURL)
	if trimmed == "" {
		return "", fmt.Errorf("base URL is empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base URL %q: scheme and host are required", rawBaseURL)
	}

	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid base URL %q: host is required", rawBaseURL)
	}

	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}

	contextPath := normalizeContextPath(parsed.Path)

	return fmt.Sprintf("%s://%s%s", scheme, formatAuthority(host, port), contextPath), nil
}

func BuildReference(mode Mode, rawBaseURL, email string) (Reference, error) {
	baseURL, err := NormalizeBaseURL(rawBaseURL)
	if err != nil {
		return Reference{}, err
	}

	ref := Reference{
		ServiceID: ServiceID,
		BaseURL:   baseURL,
	}

	switch mode {
	case ModeBearer:
		ref.AccountKey = fmt.Sprintf("bearer|%s", baseURL)
	case ModeBasic:
		trimmedEmail := strings.TrimSpace(email)
		if trimmedEmail == "" {
			return Reference{}, fmt.Errorf("email is required for basic auth store lookups")
		}
		ref.AccountKey = fmt.Sprintf("basic|%s|%s", baseURL, trimmedEmail)
	default:
		return Reference{}, fmt.Errorf("unsupported auth store mode %q", mode)
	}

	return ref, nil
}

func LookupSecret(store Store, mode Mode, rawBaseURL, email string) (string, Reference, error) {
	ref, err := BuildReference(mode, rawBaseURL, email)
	if err != nil {
		return "", Reference{}, err
	}

	if store == nil {
		store = DefaultStore()
	}

	secret, err := store.Get(ref.ServiceID, ref.AccountKey)
	if err != nil {
		return "", ref, classifyLookupError(err)
	}

	return normalizeStoredSecret(secret), ref, nil
}

func classifyLookupError(err error) error {
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrSecretNotFound
	}

	return &StoreAccessError{Cause: err}
}

func normalizeStoredSecret(secret string) string {
	raw := []byte(secret)
	if len(raw) < 2 || len(raw)%2 != 0 {
		return secret
	}

	if !strings.ContainsRune(secret, '\x00') && utf8.ValidString(secret) && !hasUTF16LEBOM(raw) {
		return secret
	}

	decoded, ok := decodeUTF16LE(raw)
	if !ok {
		return secret
	}

	return decoded
}

func hasUTF16LEBOM(raw []byte) bool {
	return len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe
}

func decodeUTF16LE(raw []byte) (string, bool) {
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i < len(raw); i += 2 {
		units = append(units, binary.LittleEndian.Uint16(raw[i:i+2]))
	}

	if len(units) > 0 && units[0] == 0xfeff {
		units = units[1:]
	}

	for len(units) > 0 && units[len(units)-1] == 0 {
		units = units[:len(units)-1]
	}

	if len(units) == 0 {
		return "", false
	}

	decodedRunes := utf16.Decode(units)
	for _, r := range decodedRunes {
		if r == utf8.RuneError || r == '\x00' {
			return "", false
		}
	}

	return string(decodedRunes), true
}

func normalizeContextPath(rawPath string) string {
	if rawPath == "" || rawPath == "/" {
		return ""
	}

	cleanPath := path.Clean(rawPath)
	if cleanPath == "." || cleanPath == "/" {
		return ""
	}

	return strings.TrimSuffix(cleanPath, "/")
}

func formatAuthority(host, port string) string {
	if port == "" {
		if strings.Contains(host, ":") {
			return "[" + host + "]"
		}
		return host
	}

	return net.JoinHostPort(host, port)
}
