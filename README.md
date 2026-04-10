# squadron-plugin-devin

A [Squadron](https://github.com/mlund01/squadron-sdk) plugin that integrates [Devin AI](https://devin.ai) for automated pull request QA, code review, and code development.

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

### `code_develop`

Develops code on a repository. Devin clones the repo, implements the requested changes, runs tests, and opens a pull request with the completed work.

Use this for:
- Feature development
- Bug fixes
- Refactoring
- Any code changes on a repository

Devin will follow existing code conventions, add or update tests, and open a PR with a detailed description.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `repo_url` | string | yes | Full URL of the GitHub repository (e.g. `https://github.com/org/repo`) |
| `task` | string | yes | Description of the development task to perform |
| `branch` | string | no | Branch name for Devin to create. If omitted, Devin chooses an appropriate name. |
| `instructions` | string | no | Additional context, constraints, or coding guidelines |

## Prerequisites

- Go 1.23+
- A [Devin AI](https://devin.ai) account with API access
- Devin must already have access to the GitHub repos you want to review or develop on

## Setup

```bash
# Clone the plugin project
git clone git@github.com:ericlakich/squadron-plugin-devin.git

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
    api_key              = "<your-devin-api-key>"
    poll_timeout_minutes = "60"
  }
}
```

Then attach the tools to an agent:

```hcl
agent "reviewer" {
  model = models.anthropic.claude_sonnet_4
  tools = [plugins.devin.code_qa, plugins.devin.code_review, plugins.devin.code_develop]
}
```

### Settings

| Setting | Required | Description |
|---------|----------|-------------|
| `api_key` | yes | Your Devin AI API key. Obtain from your Devin account settings. |
| `poll_timeout_minutes` | no | Maximum time in minutes to wait for a Devin session to complete. Defaults to `60`. Increase for long-running development tasks. |

## How It Works

1. The agent invokes a tool (`code_qa`, `code_review`, or `code_develop`) with the required parameters.
2. The plugin creates a new Devin session via the [Devin API](https://docs.devin.ai/api-reference/overview) with a task-specific prompt.
3. The plugin polls the session status every 15 seconds (up to `poll_timeout_minutes`, default 60) until Devin finishes.
4. The final result is returned as a text summary.

For `code_review`, Devin also posts inline review comments directly on the GitHub PR during its session.

For `code_develop`, Devin clones the repo, implements the changes, and opens a pull request. The PR link is included in the result.

All tools respect context cancellation, so Squadron can terminate long-running sessions cleanly.

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
