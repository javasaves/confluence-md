package commands

import (
	"fmt"
	"strings"

	"github.com/jackchuka/confluence-md/internal/confluence"
	"github.com/spf13/cobra"
)

type authOptions struct {
	APIKey    string
	Email     string
	BasicAuth bool
}

func (a *authOptions) InitFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&a.APIKey, "api-token", "t", "", "Bearer token or Basic password/token (required)")
	cmd.Flags().StringVarP(&a.Email, "email", "e", "", "Username/email for Basic auth; ignored for Bearer auth")
	cmd.Flags().BoolVar(&a.BasicAuth, "basic-auth", false, "Use HTTP Basic auth instead of the default Bearer token")
}

func (a authOptions) Validate() error {
	if strings.TrimSpace(a.APIKey) == "" {
		return fmt.Errorf("missing required flag: --api-token")
	}

	if a.BasicAuth && strings.TrimSpace(a.Email) == "" {
		return fmt.Errorf("missing required flag: --email when --basic-auth is set")
	}

	return nil
}

func (a authOptions) AuthConfig() confluence.AuthConfig {
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
	cmd.Flags().StringVarP(&c.OutputDir, "output", "o", "./output", "Output directory")
	cmd.Flags().StringVar(&c.OutputNameTemplate, "output-name-template", "", "Go template for output filename; available data: {{ .Page.* }}, {{ .SlugTitle }}, {{ .LabelNames }}")
}
