package commands

import (
	"testing"

	"github.com/javasaves/confluence-md/internal/confluence"
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

	err := opts.Validate()
	if err == nil {
		t.Fatal("expected basic auth without username/email to fail validation")
	}

	if got, want := err.Error(), "missing required flag: --email when --basic-auth is set"; got != want {
		t.Fatalf("unexpected error %q, want %q", got, want)
	}
}

func TestAuthOptionsValidateKeepsLegacyTokenFirstError(t *testing.T) {
	opts := authOptions{BasicAuth: true}

	err := opts.Validate()
	if err == nil {
		t.Fatal("expected missing token to fail validation")
	}

	if got, want := err.Error(), "missing required flag: --api-token"; got != want {
		t.Fatalf("unexpected error %q, want %q", got, want)
	}
}

func TestAuthOptionsAuthConfigUsesBasicMode(t *testing.T) {
	opts := authOptions{
		APIKey:    "secret",
		Email:     " alice ",
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
