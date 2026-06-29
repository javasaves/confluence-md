package commands

import (
	"fmt"
	"os"

	"github.com/jackchuka/confluence-md/internal/confluence"
	"github.com/jackchuka/confluence-md/internal/converter"
	"github.com/spf13/cobra"
)

// pageCmd represents the page command
var pageCmd = &cobra.Command{
	Use:   "page",
	Short: "Convert a single Confluence page to Markdown",
	Long: `Convert a single Confluence page to Markdown format.

Provide the page URL and your authentication credentials to download and convert the page.
The converted content is saved to an output directory with images in an assets folder.

Examples:
  # Convert using Bearer auth (default)
  confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title --api-token your-bearer-token

  # Convert using Basic auth
  confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title --basic-auth --email john.doe@company.com --api-token your-api-token

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

	OutputNamer converter.OutputNamer
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

	if err := pageOpts.authOptions.Validate(); err != nil {
		return fmt.Errorf("invalid authentication options: %w", err)
	}
	pageURL := args[0]

	// Extract base URL from page URL
	pageInfo, err := urlToPageInfo(pageURL)
	if err != nil {
		return fmt.Errorf("invalid Confluence URL: %w", err)
	}

	namer, err := buildOutputNamer(pageOpts.OutputNameTemplate)
	if err != nil {
		return fmt.Errorf("invalid output name template: %w", err)
	}
	pageOpts.OutputNamer = namer

	// Create Confluence client
	client := confluence.NewClient(pageInfo.BaseURL, pageOpts.authOptions.AuthConfig())

	page, err := client.GetPage(pageInfo.PageID)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(pageOpts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use shared conversion pipeline
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
