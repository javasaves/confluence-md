package commands

import (
	"fmt"
	"os"

	"github.com/javasaves/confluence-md/internal/confluence"
	confluenceModel "github.com/javasaves/confluence-md/internal/confluence/model"
	"github.com/javasaves/confluence-md/internal/converter"
	"github.com/spf13/cobra"
)

// pageCmd represents the page command
var pageCmd = &cobra.Command{
	Use:   "page",
	Short: "Convert a single Confluence page to Markdown",
	Long: `Convert a single Confluence page to Markdown format.

Provide the page URL and either manual authentication flags or OS credential store flags
to download and convert the page.
The converted content is saved to an output directory with images in an assets folder.

Examples:
  # Convert using manual Bearer auth (default)
  confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title --api-token your-bearer-token

  # Convert using manual Basic auth
  confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title --basic-auth --email john.doe@company.com --api-token your-api-token

  # Convert using a Bearer token stored in the OS credential store
  confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title --bearer-auth-store

  # Convert using a Basic auth secret stored in the OS credential store
  confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title --basic-auth-store --email john.doe@company.com

  # Convert to custom directory
  confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title --api-token your-bearer-token --output ./docs

  # Convert without downloading images
  confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title --api-token your-bearer-token --download-images=false`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return runPage(cmd, args)
	},
}

var pageOpts PageOptions

type PageOptions struct {
	authOptions
	commonOptions

	OutputNamer   converter.OutputNamer
	SourcePageURL string
}

func init() {
	rootCmd.AddCommand(pageCmd)

	pageOpts.authOptions.InitFlags(pageCmd)
	pageOpts.commonOptions.InitFlags(pageCmd)
}

func runPage(_ *cobra.Command, args []string) error {
	// Get required flags
	if len(args) < 1 {
		return fmt.Errorf("missing required argument: page URL")
	}

	pageURL := args[0]

	var (
		pageInfo confluenceModel.PageURLInfo
		err      error
	)

	if pageOpts.authOptions.usesSecretStore() {
		pageInfo, err = urlToPageInfo(pageURL)
		if err != nil {
			return fmt.Errorf("invalid Confluence URL: %w", err)
		}
	}

	if err := pageOpts.authOptions.Validate(); err != nil {
		return fmt.Errorf("invalid authentication options: %w", err)
	}

	if pageInfo.BaseURL == "" {
		// Manual auth keeps the legacy order: validate flags before parsing the page URL.
		pageInfo, err = urlToPageInfo(pageURL)
		if err != nil {
			return fmt.Errorf("invalid Confluence URL: %w", err)
		}
	}

	namer, err := buildOutputNamer(pageOpts.OutputNameTemplate)
	if err != nil {
		return fmt.Errorf("invalid output name template: %w", err)
	}
	pageOpts.OutputNamer = namer

	// Create Confluence client
	authConfig, err := pageOpts.authOptions.Resolve(pageInfo.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid authentication options: %w", err)
	}

	client := confluence.NewClient(pageInfo.BaseURL, authConfig)

	pageInfo, err = ensurePageID(client, pageInfo)
	if err != nil {
		return fmt.Errorf("failed to resolve page ID: %w", err)
	}

	page, err := client.GetPage(pageInfo.PageID)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(pageOpts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use shared conversion pipeline
	pageOpts.SourcePageURL = pageURL
	result := convertSinglePage(
		client,
		page,
		pageInfo.BaseURL,
		pageOpts,
	)

	// Print results
	printConversionResult(result)

	if !result.Success {
		return fmt.Errorf("conversion failed: %v", result.Error)
	}

	return nil
}
