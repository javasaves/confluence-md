package commands

import (
	"testing"

	"github.com/jackchuka/confluence-md/internal/confluence"
)

func TestAuthOptionsValidateBearerRequiresOnlyToken(t *testing.T) {
	opts := authOptions{APIKey: "pat-token"}

	if err := opts.Validate(); err != nil {
		t.Fatalf("expected bearer auth config to validate, got %v", err)
	}
}

func TestAuthOptionsValidateBasicRequiresUsername(t *testing.T) {
	opts := authOptions{
		APIKey:    "secret",
		BasicAuth: true,
	}

	if err := opts.Validate(); err == nil {
		t.Fatal("expected basic auth without username/email to fail validation")
	}
}

func TestAuthOptionsAuthConfigUsesBasicMode(t *testing.T) {
	opts := authOptions{
		APIKey:    "secret",
		Email:     "alice",
		BasicAuth: true,
	}

	auth := opts.AuthConfig()
	if auth.Mode != confluence.AuthModeBasic {
		t.Fatalf("expected auth mode %q, got %q", confluence.AuthModeBasic, auth.Mode)
	}

	if auth.Username != "alice" {
		t.Fatalf("expected username to be propagated, got %q", auth.Username)
	}
}
