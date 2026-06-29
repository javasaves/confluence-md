# confluence-md

[![Test](https://github.com/jackchuka/confluence-md/workflows/Test/badge.svg)](https://github.com/jackchuka/confluence-md/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/jackchuka/confluence-md)](https://goreportcard.com/report/github.com/jackchuka/confluence-md)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool to convert Confluence pages to Markdown format with a single command. Supports images, tables, lists, and various macros (**yes, even mermaid diagrams!**).

## Features

- Convert single Confluence pages to Markdown
- Convert entire page trees with hierarchical structure
- Download and embed images from Confluence pages
- Support for Confluence Cloud and self-hosted Confluence API authentication
- Enhanced support for Confluence-specific elements (user references, status badges, time elements)
- Clean, readable Markdown output
- Cross-platform support (Linux, macOS, Windows)

## Installation

### Homebrew

```bash
brew install jackchuka/tap/confluence-md
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/jackchuka/confluence-md/releases).

### From Source

```bash
go install github.com/jackchuka/confluence-md/cmd/confluence-md@latest
```

## Usage

### Authentication

`confluence-md` uses **Bearer auth by default**.

There are two explicit authentication flows:

1. Manual flags: pass the secret on the command line exactly as before.
2. OS credential store: keep the secret in the system keychain and use a store flag so the CLI never receives the secret value directly.

#### Manual authentication

For manual Bearer auth (common for self-hosted Confluence with Personal Access Tokens), you'll need:

- A bearer token / PAT

For manual Basic auth (for example Confluence Cloud with Atlassian email + API token), you'll need:

- A username or email address
- A password, API token, or other secret expected by your Confluence instance

Manual Bearer auth example:

```bash
confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title \
  --api-token your-bearer-token
```

Manual Basic auth example:

```bash
confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title \
  --basic-auth \
  --email john.doe@company.com \
  --api-token your-api-token-here
```

#### OS credential store authentication

Use one of these flags to read the secret from the system keychain:

- `--bearer-auth-store`: read a Bearer token from the OS credential store
- `--basic-auth-store`: read a Basic auth secret from the OS credential store; `--email` is always required

The stored secret is scoped by:

- `service ID`: `github.com/javasaves/confluence-md/auth/v1`
- `account key` for Bearer: `bearer|<normalized-base-url>`
- `account key` for Basic: `basic|<normalized-base-url>|<email>`

`<normalized-base-url>` is always reduced to the Confluence origin plus context path only:

- scheme and host are lowercased
- default ports are removed (`:80` for `http`, `:443` for `https`)
- query strings and fragments are ignored
- the root path becomes empty
- context paths such as `/wiki` or `/confluence` are preserved without a trailing slash

Examples:

- `https://Example.Atlassian.Net:443/wiki/spaces/SPACE/pages/12345/Title?draft=1#frag` -> `https://example.atlassian.net/wiki`
- `https://wiki.company.local/spaces/SPACE/pages/12345/Title` -> `https://wiki.company.local`

Stored Bearer auth example:

```bash
confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title \
  --bearer-auth-store
```

Stored Basic auth example:

```bash
confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title \
  --basic-auth-store \
  --email john.doe@company.com
```

##### Creating the keychain entry manually

Use the exact `service ID` and `account key` shown above. They are also echoed back in the CLI error message when an entry is missing.

macOS:

```bash
security add-generic-password -U \
  -s "github.com/javasaves/confluence-md/auth/v1" \
  -a "bearer|https://confluence.example.com" \
  -w "your-bearer-token"

security add-generic-password -U \
  -s "github.com/javasaves/confluence-md/auth/v1" \
  -a "basic|https://example.atlassian.net/wiki|john.doe@company.com" \
  -w "your-basic-secret"
```

Linux with Secret Service and `secret-tool`:

```bash
printf %s "your-bearer-token" | secret-tool store \
  --label="confluence-md bearer token" \
  service "github.com/javasaves/confluence-md/auth/v1" \
  username "bearer|https://confluence.example.com"

printf %s "your-basic-secret" | secret-tool store \
  --label="confluence-md basic auth secret" \
  service "github.com/javasaves/confluence-md/auth/v1" \
  username "basic|https://example.atlassian.net/wiki|john.doe@company.com"
```

Windows Credential Manager (`cmdkey` or PowerShell with the `CredentialManager` module):

```powershell
cmdkey /generic:"github.com/javasaves/confluence-md/auth/v1:bearer|https://confluence.example.com" /user:"bearer|https://confluence.example.com" /pass:"your-bearer-token"

cmdkey /generic:"github.com/javasaves/confluence-md/auth/v1:basic|https://example.atlassian.net/wiki|john.doe@company.com" /user:"basic|https://example.atlassian.net/wiki|john.doe@company.com" /pass:"your-basic-secret"

# Or, with the CredentialManager module:
New-StoredCredential -Target "github.com/javasaves/confluence-md/auth/v1:basic|https://example.atlassian.net/wiki|john.doe@company.com" `
  -UserName "basic|https://example.atlassian.net/wiki|john.doe@company.com" `
  -Password "your-basic-secret" `
  -Type Generic -Persist LocalMachine
```

For Windows Credential Manager, use a `Generic Credential` whose `Target/Address` is `<service ID>:<account key>` and whose `UserName` is `<account key>`. The secret itself must be stored in the password field.

Some Windows tools store the password blob as UTF-16LE instead of plain bytes. `confluence-md` detects and decodes UTF-16LE secrets automatically when reading from Windows Credential Manager, so credentials created with `cmdkey` or PowerShell tools can still work correctly.

If the stored entry is missing, the CLI reports an English error that includes the exact `service ID` and `account key`, for example:

```text
no stored secret found in the OS credential store for service ID "github.com/javasaves/confluence-md/auth/v1" and account key "bearer|https://confluence.example.com". For Windows Credential Manager, create a Generic Credential with Target/Address "github.com/javasaves/confluence-md/auth/v1:bearer|https://confluence.example.com", UserName "bearer|https://confluence.example.com", and the secret in the password field
```

If the system keychain is locked, unavailable, or not configured, the CLI reports a different English error without the manual creation hint, for example:

```text
failed to access the OS credential store while looking up service ID "github.com/javasaves/confluence-md/auth/v1" and account key "bearer|https://confluence.example.com". For Windows Credential Manager, the expected Target/Address is "github.com/javasaves/confluence-md/auth/v1:bearer|https://confluence.example.com" and the expected UserName is "bearer|https://confluence.example.com": keychain is locked
```

**Troubleshooting on Windows:** If you see `net/http: invalid header field value for "Authorization"`, the stored secret usually contains control characters or was written in UTF-16LE form. `confluence-md` decodes UTF-16LE secrets automatically, but if the error persists, recreate the `Generic Credential` and make sure:

- `Target/Address` exactly matches the value shown by the CLI error
- `UserName` exactly matches the value shown by the CLI error
- the token or password is stored in the password field only

### Convert a Single Page

```bash
confluence-md page <page-url> --api-token your-bearer-token
```

Example:

```bash
confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title \
  --api-token your-bearer-token
```

Confluence Cloud / Basic auth example:

```bash
confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title \
  --basic-auth \
  --email john.doe@company.com \
  --api-token your-api-token-here
```

Use a Bearer token from the OS credential store:

```bash
confluence-md page https://confluence.example.com/spaces/SPACE/pages/12345/Title \
  --bearer-auth-store
```

Use a Basic auth secret from the OS credential store:

```bash
confluence-md page https://example.atlassian.net/wiki/spaces/SPACE/pages/12345/Title \
  --basic-auth-store \
  --email john.doe@company.com
```

### Convert a Page Tree

Convert an entire page hierarchy:

```bash
confluence-md tree <page-url> --api-token your-bearer-token
```

With the OS credential store:

```bash
confluence-md tree <page-url> --bearer-auth-store
```

### Convert HTML Files

Convert Confluence HTML directly without API access (useful for testing or working with exported HTML):

```bash
# Convert from file
confluence-md html page.html -o output.md

# Convert from stdin
cat page.html | confluence-md html -o output.md

# Output to stdout
confluence-md html page.html
```

### Common Options

- `--api-token, -t`: Bearer token or Basic password/token (**required for manual auth**)
- `--email, -e`: Username/email for Basic auth; required for `--basic-auth` and `--basic-auth-store`
- `--basic-auth`: Use HTTP Basic auth instead of the default Bearer auth
- `--bearer-auth-store`: Read the Bearer token from the OS credential store
- `--basic-auth-store`: Read the Basic auth secret from the OS credential store
- `--output, -o`: Output directory (default: current directory)
- `--output-name-template`: Go template for the markdown filename (see below)
- `--download-images`: Download images from Confluence (default: true)
- `--image-folder`: Folder to save images (default: `assets`)
- `--include-metadata`: Include page metadata in the Markdown front matter (default: true)

### Examples

```bash
# Convert to a specific directory using Bearer auth
confluence-md page <page-url> --api-token token --output ./docs

# Prefix filenames with the last updated date (YYYY-MM-DD-title.md)
confluence-md page <page-url> \
  --api-token token \
  --output-name-template "{{ .Page.UpdatedAt.Format \"2006-01-02\" }}-{{ .SlugTitle }}"

# Convert using Basic auth (for example Confluence Cloud)
confluence-md page <page-url> \
  --basic-auth \
  --email user@example.com \
  --api-token token

# Convert using the OS credential store with Bearer auth
confluence-md page <page-url> --bearer-auth-store

# Convert using the OS credential store with Basic auth
confluence-md page <page-url> \
  --basic-auth-store \
  --email user@example.com

# Convert without downloading images
confluence-md page <page-url> --api-token token --download-images=false

# Convert entire page tree
confluence-md tree <page-url> --api-token token --output ./wiki
```

Page URLs with no context path (for example `https://wiki.company.local/spaces/...`), with `/wiki`, and with other context paths such as `/confluence` are supported.

### Output name templates

The `--output-name-template` flag accepts a Go text/template string. Templates can reference:

- `{{ .Page }}` – the full Confluence page object (e.g. `{{ .Page.UpdatedAt.Format "2006-01-02" }}`)
  - `{{ .Page.Title }}` – the original page title
  - `{{ .Page.ID }}` – the Confluence page ID
  - `{{ .Page.SpaceKey }}` – the Confluence space key
  - see ConfluencePage struct for more fields
- `{{ .SlugTitle }}` – the default slugified title (e.g. `sample-page`)

Additionally, you can use the following helper functions:

- `{{ slug <string> }}` – slugifies a string (e.g. `Sample Page` → `sample-page`)

If the rendered filename omits an extension, `.md` is appended automatically.

## Supported Confluence Elements

### Basic Elements

| Element             | Confluence Tag             | Conversion                                                              |
| ------------------- | -------------------------- | ----------------------------------------------------------------------- |
| **Images**          | `ac:image`                 | Downloaded and converted to local markdown image references             |
| **Emoticons**       | `ac:emoticon`              | Converted to emoji fallback or shortnames                               |
| **Tables**          | Standard HTML tables       | Full table support with proper markdown formatting                      |
| **Lists**           | Standard HTML lists        | Nested lists with proper indentation                                    |
| **User Links**      | `ac:link` + `ri:user`      | Converted to `@DisplayName` (or `@user(account-id)` if name not cached) |
| **Time Elements**   | `<time>`                   | Datetime attribute extracted and displayed                              |
| **Inline Comments** | `ac:inline-comment-marker` | Text preserved with comment reference                                   |
| **Placeholders**    | `ac:placeholder`           | Converted to HTML comments                                              |

### Macros (`ac:structured-macro`)

| Macro               | Status                      | Conversion                                                          |
| ------------------- | --------------------------- | ------------------------------------------------------------------- |
| **`info`**          | ✅ Fully Supported          | Converted to blockquote with ℹ️ Info prefix                         |
| **`warning`**       | ✅ Fully Supported          | Converted to blockquote with ⚠️ Warning prefix                      |
| **`note`**          | ✅ Fully Supported          | Converted to blockquote with 📝 Note prefix                         |
| **`tip`**           | ✅ Fully Supported          | Converted to blockquote with 💡 Tip prefix                          |
| **`code`**          | ✅ Fully Supported          | Converted to markdown code blocks with optional `**title**` caption |
| **`jira`**          | ⚠️ Partially Supported      | Emits the issue key, linking it when Jira URL can be derived        |
| **`mermaid-cloud`** | ✅ Fully Supported          | Converted to mermaid code blocks                                    |
| **`expand`**        | ✅ Fully Supported          | Content extracted and rendered directly                             |
| **`details`**       | ✅ Fully Supported          | Content extracted and rendered directly                             |
| **`status`**        | ✅ Fully Supported          | Converted to emoji badges (🔴 **S1**, 🟡, 🟢, 🔵, ⚪)               |
| **`toc`**           | ⚠️ Partially Supported      | Converted to `<!-- Table of Contents -->` comment                   |
| **`children`**      | ⚠️ Partially Supported      | Converted to `<!-- Child Pages -->` comment                         |
| **Other macros**    | Plan to support per request | Converted to visible `**Unsupported macro:** \`{name}\`` markers    |

If a Confluence `code` macro includes a `title` parameter, `confluence-md` writes a bold caption line such as `**main.go**` above the fenced code block. When converting pages via the `page` or `tree` commands, `jira` issue keys are linked by deriving a Jira base URL from the Confluence base URL when possible, for example `confluence.example.com` -> `jira.example.com` and `example.atlassian.net/wiki` -> `example.atlassian.net`. Unsupported macros such as `drawio`, and unsupported `jira` variants without a usable `key`, remain visible in Markdown output instead of being hidden inside HTML comments.

### User Name Resolution

User references (`@user`) are automatically resolved to display names when converting pages via the `page` or `tree` commands

**Note:** When using the `html` command (without Confluence API access), user names cannot be resolved and will always display as `@user(account-id)`.

## Output Structure

The tool creates:

- Markdown files (.md) for each page
- An `assets/` directory containing downloaded images
- Hierarchical directory structure for page trees

## Development

### Prerequisites

- Go 1.24.4 or later

### Building

```bash
git clone https://github.com/jackchuka/confluence-md.git
cd confluence-md
go build -o confluence-md cmd/confluence-md/main.go
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run tests and linting
6. Submit a pull request

## Support

For issues and feature requests, please use the [GitHub issue tracker](https://github.com/jackchuka/confluence-md/issues).
