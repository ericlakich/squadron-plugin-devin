# squadron-plugin-devin

A [Squadron](https://github.com/mlund01/squadron-sdk) plugin that integrates [Devin AI](https://devin.ai) for automated pull request QA and code review.

## Tools

### `code_qa`

Performs a full QA review of a pull request. Devin checks out the PR branch, analyzes changes, runs existing tests, and returns a comprehensive summary.

The QA review covers:
- Bug detection, edge cases, and logic errors
- Error handling adequacy
- Test execution and failure reporting
- Missing test coverage for new/changed code
- Regression risks in related functionality
- Alignment with PR description and linked issues
- Performance concerns

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `pr_url` | string | yes | Full URL of the GitHub PR (e.g. `https://github.com/org/repo/pull/123`) |
| `instructions` | string | no | Additional instructions or focus areas for the QA review |

### `code_review`

Performs a full code review of a pull request. Devin reviews the diff, posts inline comments directly on the GitHub PR, and submits an overall review summary.

The code review covers:
- Every changed file in the PR diff
- Code quality, readability, and maintainability
- Correctness and potential bugs
- Security concerns and vulnerabilities
- Adherence to best practices and coding conventions
- Improvement suggestions

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `pr_url` | string | yes | Full URL of the GitHub PR (e.g. `https://github.com/org/repo/pull/123`) |
| `instructions` | string | no | Additional instructions or focus areas for the code review |

## Prerequisites

- Go 1.23+
- A [Devin AI](https://devin.ai) account with API access
- Devin must already have access to the GitHub repos you want to review
- The [squadron-sdk](https://github.com/mlund01/squadron-sdk) cloned as a sibling directory (see below)

## Setup

```bash
# Clone the plugin and the SDK side by side
git clone git@github.com:ericlakich/squadron-plugin-devin.git
git clone git@github.com:mlund01/squadron-sdk.git

# Directory layout should be:
#   squadron-plugin-devin/
#   squadron-sdk/

# Build
cd squadron-plugin-devin
go mod tidy
go build -o plugin .

# Install into Squadron's plugin directory
mkdir -p ~/.squadron/plugins/devin/local
cp plugin ~/.squadron/plugins/devin/local/plugin
```

## Configuration

Add the plugin to your Squadron HCL config:

```hcl
plugin "devin" {
  version = "local"
  settings = {
    api_key = "<your-devin-api-key>"
  }
}
```

Then attach the tools to an agent:

```hcl
agent "reviewer" {
  model = models.anthropic.claude_sonnet_4
  tools = [plugins.devin.code_qa, plugins.devin.code_review]
}
```

### Settings

| Setting | Required | Description |
|---------|----------|-------------|
| `api_key` | yes | Your Devin AI API key. Obtain from your Devin account settings. |

## How It Works

1. The agent invokes `code_qa` or `code_review` with a PR URL.
2. The plugin creates a new Devin session via the [Devin API](https://docs.devin.ai/api-reference/overview) with a task-specific prompt.
3. The plugin polls the session status every 15 seconds (up to 30 minutes) until Devin finishes.
4. The final result is returned as a text summary.

For `code_review`, Devin also posts inline review comments directly on the GitHub PR during its session.

Both tools respect context cancellation, so Squadron can terminate long-running sessions cleanly.

## Project Structure

```
squadron-plugin-devin/
  main.go          # Entry point - registers the plugin with Squadron
  plugin.go        # ToolProvider implementation, tool definitions, prompt builders
  devin/
    client.go      # HTTP client for the Devin AI API (v1)
  go.mod
  go.sum
```

## License

See [LICENSE](LICENSE) for details.
