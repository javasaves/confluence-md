package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/javasaves/confluence-md/internal/confluence"
	"github.com/javasaves/confluence-md/internal/credentials"
	"github.com/spf13/cobra"
)

type authOptions struct {
	APIKey          string
	Email           string
	BasicAuth       bool
	BearerAuthStore bool
	BasicAuthStore  bool

	secretStore credentials.Store
}

func (a authOptions) normalized() authOptions {
	a.Email = strings.TrimSpace(a.Email)
	return a
}

func (a *authOptions) InitFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&a.APIKey, "api-token", "t", "", "Bearer token or Basic password/token (required for manual auth)")
	cmd.Flags().StringVarP(&a.Email, "email", "e", "", "Username/email for Basic auth; required for --basic-auth and --basic-auth-store")
	cmd.Flags().BoolVar(&a.BasicAuth, "basic-auth", false, "Use HTTP Basic auth instead of the default Bearer token")
	cmd.Flags().BoolVar(&a.BearerAuthStore, "bearer-auth-store", false, "Read the Bearer token from the OS credential store")
	cmd.Flags().BoolVar(&a.BasicAuthStore, "basic-auth-store", false, "Read the Basic auth secret from the OS credential store (requires --email)")
}

func (a authOptions) Validate() error {
	a = a.normalized()

	if err := a.validateModeSelection(); err != nil {
		return err
	}

	if a.usesSecretStore() {
		if a.BasicAuthStore && a.Email == "" {
			return fmt.Errorf("missing required flag: --email when --basic-auth-store is set")
		}

		return nil
	}

	return a.validateManual()
}

func (a authOptions) Resolve(baseURL string) (confluence.AuthConfig, error) {
	a = a.normalized()

	if err := a.Validate(); err != nil {
		return confluence.AuthConfig{}, err
	}

	switch {
	case a.BearerAuthStore:
		return a.resolveStored(baseURL, credentials.ModeBearer)
	case a.BasicAuthStore:
		return a.resolveStored(baseURL, credentials.ModeBasic)
	default:
		return a.AuthConfig(), nil
	}
}

func (a authOptions) AuthConfig() confluence.AuthConfig {
	a = a.normalized()

	if a.BasicAuth {
		return confluence.AuthConfig{
			Mode:     confluence.AuthModeBasic,
			Username: a.Email,
			Secret:   a.APIKey,
		}
	}

	return confluence.AuthConfig{
		Mode:   confluence.AuthModeBearer,
		Secret: a.APIKey,
	}
}

func (a authOptions) validateManual() error {
	if strings.TrimSpace(a.APIKey) == "" {
		return fmt.Errorf("missing required flag: --api-token")
	}

	if a.BasicAuth && a.Email == "" {
		return fmt.Errorf("missing required flag: --email when --basic-auth is set")
	}

	return nil
}

func (a authOptions) validateModeSelection() error {
	if a.BearerAuthStore && a.BasicAuthStore {
		return fmt.Errorf("flags --bearer-auth-store and --basic-auth-store are mutually exclusive")
	}

	if !a.usesSecretStore() {
		return nil
	}

	storeFlag := a.selectedStoreFlag()

	if strings.TrimSpace(a.APIKey) != "" {
		return fmt.Errorf("flag --api-token cannot be used with %s", storeFlag)
	}

	if a.BasicAuth {
		return fmt.Errorf("flag --basic-auth cannot be used with %s", storeFlag)
	}

	return nil
}

func (a authOptions) resolveStored(baseURL string, mode credentials.Mode) (confluence.AuthConfig, error) {
	a = a.normalized()

	secret, ref, err := credentials.LookupSecret(a.store(), mode, baseURL, a.Email)
	if err != nil {
		if errors.Is(err, credentials.ErrSecretNotFound) {
			return confluence.AuthConfig{}, fmt.Errorf(
				"no stored secret found in the OS credential store for service ID %q and account key %q. For Windows Credential Manager, create a Generic Credential with Target/Address %q, UserName %q, and the secret in the password field",
				ref.ServiceID,
				ref.AccountKey,
				ref.WindowsTargetName(),
				ref.WindowsUserName(),
			)
		}

		if errors.Is(err, credentials.ErrSecretStoreUnavailable) {
			return confluence.AuthConfig{}, fmt.Errorf(
				"failed to access the OS credential store while looking up service ID %q and account key %q. For Windows Credential Manager, the expected Target/Address is %q and the expected UserName is %q: %v",
				ref.ServiceID,
				ref.AccountKey,
				ref.WindowsTargetName(),
				ref.WindowsUserName(),
				err,
			)
		}

		return confluence.AuthConfig{}, fmt.Errorf("failed to resolve stored authentication: %w", err)
	}

	if mode == credentials.ModeBasic {
		return confluence.AuthConfig{
			Mode:     confluence.AuthModeBasic,
			Username: a.Email,
			Secret:   secret,
		}, nil
	}

	return confluence.AuthConfig{
		Mode:   confluence.AuthModeBearer,
		Secret: secret,
	}, nil
}

func (a authOptions) usesSecretStore() bool {
	return a.BearerAuthStore || a.BasicAuthStore
}

func (a authOptions) selectedStoreFlag() string {
	if a.BasicAuthStore {
		return "--basic-auth-store"
	}

	return "--bearer-auth-store"
}

func (a authOptions) store() credentials.Store {
	if a.secretStore != nil {
		return a.secretStore
	}

	return credentials.DefaultStore()
}

type commonOptions struct {
	DownloadImages     bool
	ImageFolder        string
	IncludeMetadata    bool
	OutputDir          string
	OutputNameTemplate string
}

func (c *commonOptions) InitFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&c.DownloadImages, "download-images", true, "Download images locally")
	cmd.Flags().StringVar(&c.ImageFolder, "image-folder", "assets", "Folder for downloaded images")
	cmd.Flags().BoolVar(&c.IncludeMetadata, "include-metadata", true, "Include YAML frontmatter")
	cmd.Flags().StringVarP(&c.OutputDir, "output", "o", ".", "Output directory")
	cmd.Flags().StringVar(&c.OutputNameTemplate, "output-name-template", "", "Go template for output filename; available data: {{ .Page.* }}, {{ .SlugTitle }}, {{ .LabelNames }}")
}
