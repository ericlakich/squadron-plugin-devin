package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	squadron "github.com/mlund01/squadron-sdk"
	"github.com/ericlakich/squadron-plugin-devin/devin"
)

// tools defines the metadata for all tools provided by this plugin.
var tools = map[string]*squadron.ToolInfo{
	"code_qa": {
		Name: "code_qa",
		Description: "Perform a full QA review of a pull request using Devin AI. " +
			"Devin will check out the PR, analyze the changes, run tests if applicable, " +
			"and return a comprehensive QA summary including potential issues, test coverage gaps, " +
			"and suggestions for improvement.",
		Schema: squadron.Schema{
			Type: squadron.TypeObject,
			Properties: squadron.PropertyMap{
				"pr_url": {
					Type:        squadron.TypeString,
					Description: "The full URL of the GitHub pull request to QA (e.g. https://github.com/org/repo/pull/123)",
				},
				"instructions": {
					Type:        squadron.TypeString,
					Description: "Optional additional instructions or focus areas for the QA review",
				},
			},
			Required: []string{"pr_url"},
		},
	},
	"code_review": {
		Name: "code_review",
		Description: "Perform a full code review of a pull request using Devin AI. " +
			"Devin will review the PR diff, leave inline comments on the GitHub PR, " +
			"and provide an overall review summary covering code quality, correctness, " +
			"security concerns, and adherence to best practices.",
		Schema: squadron.Schema{
			Type: squadron.TypeObject,
			Properties: squadron.PropertyMap{
				"pr_url": {
					Type:        squadron.TypeString,
					Description: "The full URL of the GitHub pull request to review (e.g. https://github.com/org/repo/pull/123)",
				},
				"instructions": {
					Type:        squadron.TypeString,
					Description: "Optional additional instructions or focus areas for the code review",
				},
			},
			Required: []string{"pr_url"},
		},
	},
	"code_develop": {
		Name: "code_develop",
		Description: "Develop code on a repository using Devin AI. " +
			"Devin will clone the repo, implement the requested changes, run tests, " +
			"and open a pull request with the completed work. " +
			"Use this for feature development, bug fixes, refactoring, or any code changes.",
		Schema: squadron.Schema{
			Type: squadron.TypeObject,
			Properties: squadron.PropertyMap{
				"repo_url": {
					Type:        squadron.TypeString,
					Description: "The full URL of the GitHub repository to develop on (e.g. https://github.com/org/repo)",
				},
				"task": {
					Type:        squadron.TypeString,
					Description: "A description of the development task to perform (e.g. 'Add pagination to the /users API endpoint')",
				},
				"branch": {
					Type:        squadron.TypeString,
					Description: "Optional branch name for Devin to create. If not specified, Devin will choose an appropriate name.",
				},
				"instructions": {
					Type:        squadron.TypeString,
					Description: "Optional additional context, constraints, or coding guidelines for the task",
				},
			},
			Required: []string{"repo_url", "task"},
		},
	},
}

// Plugin implements the squadron.ToolProvider interface for Devin AI integration.
type Plugin struct {
	client      *devin.Client
	apiKey      string
	pollTimeout time.Duration
}

// Configure receives settings from the Squadron HCL config.
// Required settings:
//   - api_key: The Devin AI API key for authentication.
//
// Optional settings:
//   - poll_timeout_minutes: Maximum time in minutes to wait for a Devin session
//     to complete. Defaults to 60.
func (p *Plugin) Configure(settings map[string]string) error {
	apiKey, ok := settings["api_key"]
	if !ok || apiKey == "" {
		return fmt.Errorf("missing required setting: api_key")
	}
	p.apiKey = apiKey
	p.client = devin.NewClient(apiKey)

	p.pollTimeout = 60 * time.Minute
	if v, ok := settings["poll_timeout_minutes"]; ok && v != "" {
		minutes, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid poll_timeout_minutes %q: %w", v, err)
		}
		if minutes < 1 {
			return fmt.Errorf("poll_timeout_minutes must be at least 1, got %d", minutes)
		}
		p.pollTimeout = time.Duration(minutes) * time.Minute
	}

	return nil
}

// Call dispatches a tool invocation to the appropriate handler.
func (p *Plugin) Call(ctx context.Context, toolName string, payload string) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("plugin not configured: call Configure first")
	}

	switch toolName {
	case "code_qa":
		return p.callCodeQA(ctx, payload)
	case "code_review":
		return p.callCodeReview(ctx, payload)
	case "code_develop":
		return p.callCodeDevelop(ctx, payload)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// GetToolInfo returns metadata for a specific tool.
func (p *Plugin) GetToolInfo(toolName string) (*squadron.ToolInfo, error) {
	info, ok := tools[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
	return info, nil
}

// ListTools returns metadata for all tools provided by this plugin.
func (p *Plugin) ListTools() ([]*squadron.ToolInfo, error) {
	result := make([]*squadron.ToolInfo, 0, len(tools))
	for _, info := range tools {
		result = append(result, info)
	}
	return result, nil
}

// codeQAParams are the parameters for the code_qa tool.
type codeQAParams struct {
	PRURL        string `json:"pr_url"`
	Instructions string `json:"instructions,omitempty"`
}

// codeReviewParams are the parameters for the code_review tool.
type codeReviewParams struct {
	PRURL        string `json:"pr_url"`
	Instructions string `json:"instructions,omitempty"`
}

// codeDevelopParams are the parameters for the code_develop tool.
type codeDevelopParams struct {
	RepoURL      string `json:"repo_url"`
	Task         string `json:"task"`
	Branch       string `json:"branch,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

// callCodeQA creates a Devin session to perform QA on a PR and polls until completion.
func (p *Plugin) callCodeQA(ctx context.Context, payload string) (string, error) {
	var params codeQAParams
	if err := json.Unmarshal([]byte(payload), &params); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}
	if params.PRURL == "" {
		return "", fmt.Errorf("pr_url is required")
	}

	prompt := buildQAPrompt(params.PRURL, params.Instructions)

	session, err := p.client.CreateSession(ctx, devin.CreateSessionRequest{
		Prompt:     prompt,
		Idempotent: false,
	})
	if err != nil {
		return "", fmt.Errorf("create devin session: %w", err)
	}

	status, err := p.client.PollUntilDone(ctx, session.SessionID, 0, p.pollTimeout)
	if err != nil {
		return "", fmt.Errorf("waiting for devin session %s: %w", session.SessionID, err)
	}

	return formatQAResult(session.SessionID, session.URL, status), nil
}

// callCodeReview creates a Devin session to review a PR and polls until completion.
func (p *Plugin) callCodeReview(ctx context.Context, payload string) (string, error) {
	var params codeReviewParams
	if err := json.Unmarshal([]byte(payload), &params); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}
	if params.PRURL == "" {
		return "", fmt.Errorf("pr_url is required")
	}

	prompt := buildReviewPrompt(params.PRURL, params.Instructions)

	session, err := p.client.CreateSession(ctx, devin.CreateSessionRequest{
		Prompt:     prompt,
		Idempotent: false,
	})
	if err != nil {
		return "", fmt.Errorf("create devin session: %w", err)
	}

	status, err := p.client.PollUntilDone(ctx, session.SessionID, 0, p.pollTimeout)
	if err != nil {
		return "", fmt.Errorf("waiting for devin session %s: %w", session.SessionID, err)
	}

	return formatReviewResult(session.SessionID, session.URL, status), nil
}

// buildQAPrompt constructs the Devin prompt for a QA review.
func buildQAPrompt(prURL string, instructions string) string {
	var b strings.Builder
	b.WriteString("Perform a thorough QA review of this pull request: ")
	b.WriteString(prURL)
	b.WriteString("\n\n")
	b.WriteString("Your QA review should include:\n")
	b.WriteString("1. Check out the PR branch and understand the changes\n")
	b.WriteString("2. Identify potential bugs, edge cases, or logic errors\n")
	b.WriteString("3. Verify error handling is adequate\n")
	b.WriteString("4. Run any existing tests and note failures\n")
	b.WriteString("5. Identify missing test coverage for new or changed code\n")
	b.WriteString("6. Check for regressions in related functionality\n")
	b.WriteString("7. Verify the changes match the PR description and any linked issues\n")
	b.WriteString("8. Note any performance concerns\n\n")
	b.WriteString("Provide a detailed summary of your findings with clear categorization ")
	b.WriteString("(critical issues, warnings, suggestions, and things that look good).\n")

	if instructions != "" {
		b.WriteString("\nAdditional instructions: ")
		b.WriteString(instructions)
		b.WriteString("\n")
	}

	return b.String()
}

// buildReviewPrompt constructs the Devin prompt for a code review.
func buildReviewPrompt(prURL string, instructions string) string {
	var b strings.Builder
	b.WriteString("Perform a thorough code review of this pull request: ")
	b.WriteString(prURL)
	b.WriteString("\n\n")
	b.WriteString("Your code review should:\n")
	b.WriteString("1. Review every changed file in the PR diff\n")
	b.WriteString("2. Leave inline review comments directly on the GitHub PR for specific issues\n")
	b.WriteString("3. Evaluate code quality, readability, and maintainability\n")
	b.WriteString("4. Check for correctness and potential bugs\n")
	b.WriteString("5. Identify security concerns or vulnerabilities\n")
	b.WriteString("6. Verify adherence to best practices and coding conventions\n")
	b.WriteString("7. Suggest improvements where appropriate\n")
	b.WriteString("8. Submit your review on GitHub with an overall summary comment\n\n")
	b.WriteString("After completing the review, provide a summary of your findings.\n")

	if instructions != "" {
		b.WriteString("\nAdditional instructions: ")
		b.WriteString(instructions)
		b.WriteString("\n")
	}

	return b.String()
}

// formatQAResult formats the QA session result into a readable text summary.
func formatQAResult(sessionID, sessionURL string, status *devin.SessionStatus) string {
	var b strings.Builder
	b.WriteString("=== Devin QA Review Complete ===\n\n")
	b.WriteString(fmt.Sprintf("Session: %s\n", sessionID))
	b.WriteString(fmt.Sprintf("URL: %s\n", sessionURL))
	b.WriteString(fmt.Sprintf("Status: %s\n\n", status.StatusEnum))

	if status.Title != "" {
		b.WriteString(fmt.Sprintf("Title: %s\n\n", status.Title))
	}

	if len(status.StructuredOutput) > 0 && string(status.StructuredOutput) != "null" {
		b.WriteString("Structured Output:\n")
		// Pretty-print the structured output
		var pretty json.RawMessage
		if err := json.Unmarshal(status.StructuredOutput, &pretty); err == nil {
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			b.WriteString(string(formatted))
		} else {
			b.WriteString(string(status.StructuredOutput))
		}
		b.WriteString("\n\n")
	}

	b.WriteString("View the full Devin session for detailed findings: ")
	b.WriteString(sessionURL)
	b.WriteString("\n")

	return b.String()
}

// callCodeDevelop creates a Devin session to develop code on a repo and polls until completion.
func (p *Plugin) callCodeDevelop(ctx context.Context, payload string) (string, error) {
	var params codeDevelopParams
	if err := json.Unmarshal([]byte(payload), &params); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}
	if params.RepoURL == "" {
		return "", fmt.Errorf("repo_url is required")
	}
	if params.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	prompt := buildDevelopPrompt(params.RepoURL, params.Task, params.Branch, params.Instructions)

	session, err := p.client.CreateSession(ctx, devin.CreateSessionRequest{
		Prompt:     prompt,
		Idempotent: false,
	})
	if err != nil {
		return "", fmt.Errorf("create devin session: %w", err)
	}

	status, err := p.client.PollUntilDone(ctx, session.SessionID, 0, p.pollTimeout)
	if err != nil {
		return "", fmt.Errorf("waiting for devin session %s: %w", session.SessionID, err)
	}

	return formatDevelopResult(session.SessionID, session.URL, status), nil
}

// buildDevelopPrompt constructs the Devin prompt for a development task.
func buildDevelopPrompt(repoURL, task, branch, instructions string) string {
	var b strings.Builder
	b.WriteString("You are working on the repository: ")
	b.WriteString(repoURL)
	b.WriteString("\n\n")
	b.WriteString("Task: ")
	b.WriteString(task)
	b.WriteString("\n\n")
	b.WriteString("Please complete this development task by following these steps:\n")
	b.WriteString("1. Clone the repository and understand the existing codebase structure\n")
	b.WriteString("2. Create a new branch for your changes")
	if branch != "" {
		b.WriteString(" named: ")
		b.WriteString(branch)
	}
	b.WriteString("\n")
	b.WriteString("3. Implement the requested changes with clean, well-structured code\n")
	b.WriteString("4. Follow existing code conventions and patterns in the repository\n")
	b.WriteString("5. Add or update tests to cover the changes\n")
	b.WriteString("6. Run the existing test suite and ensure all tests pass\n")
	b.WriteString("7. Commit your changes with clear, descriptive commit messages\n")
	b.WriteString("8. Open a pull request with a detailed description of the changes\n\n")

	if instructions != "" {
		b.WriteString("Additional instructions: ")
		b.WriteString(instructions)
		b.WriteString("\n\n")
	}

	b.WriteString("When complete, provide a summary of the changes made and the PR link.\n")

	return b.String()
}

// formatDevelopResult formats the development session result into a readable text summary.
func formatDevelopResult(sessionID, sessionURL string, status *devin.SessionStatus) string {
	var b strings.Builder
	b.WriteString("=== Devin Development Complete ===\n\n")
	b.WriteString(fmt.Sprintf("Session: %s\n", sessionID))
	b.WriteString(fmt.Sprintf("URL: %s\n", sessionURL))
	b.WriteString(fmt.Sprintf("Status: %s\n\n", status.StatusEnum))

	if status.Title != "" {
		b.WriteString(fmt.Sprintf("Title: %s\n\n", status.Title))
	}

	if status.PullRequest != nil && status.PullRequest.URL != "" {
		b.WriteString(fmt.Sprintf("Pull Request: %s\n\n", status.PullRequest.URL))
	}

	if len(status.StructuredOutput) > 0 && string(status.StructuredOutput) != "null" {
		b.WriteString("Structured Output:\n")
		var pretty json.RawMessage
		if err := json.Unmarshal(status.StructuredOutput, &pretty); err == nil {
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			b.WriteString(string(formatted))
		} else {
			b.WriteString(string(status.StructuredOutput))
		}
		b.WriteString("\n\n")
	}

	b.WriteString("View the full Devin session for details: ")
	b.WriteString(sessionURL)
	b.WriteString("\n")

	return b.String()
}

// formatReviewResult formats the code review session result into a readable text summary.
func formatReviewResult(sessionID, sessionURL string, status *devin.SessionStatus) string {
	var b strings.Builder
	b.WriteString("=== Devin Code Review Complete ===\n\n")
	b.WriteString(fmt.Sprintf("Session: %s\n", sessionID))
	b.WriteString(fmt.Sprintf("URL: %s\n", sessionURL))
	b.WriteString(fmt.Sprintf("Status: %s\n\n", status.StatusEnum))

	if status.Title != "" {
		b.WriteString(fmt.Sprintf("Title: %s\n\n", status.Title))
	}

	if status.PullRequest != nil && status.PullRequest.URL != "" {
		b.WriteString(fmt.Sprintf("PR: %s\n\n", status.PullRequest.URL))
	}

	if len(status.StructuredOutput) > 0 && string(status.StructuredOutput) != "null" {
		b.WriteString("Structured Output:\n")
		var pretty json.RawMessage
		if err := json.Unmarshal(status.StructuredOutput, &pretty); err == nil {
			formatted, _ := json.MarshalIndent(pretty, "", "  ")
			b.WriteString(string(formatted))
		} else {
			b.WriteString(string(status.StructuredOutput))
		}
		b.WriteString("\n\n")
	}

	b.WriteString("Review comments have been posted directly on the GitHub PR.\n")
	b.WriteString("View the full Devin session: ")
	b.WriteString(sessionURL)
	b.WriteString("\n")

	return b.String()
}
