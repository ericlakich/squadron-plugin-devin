# Devin Code Review Skill Guide

Use the Devin plugin to perform automated code reviews of pull requests. Devin posts inline comments directly on the GitHub PR.

## Workflow

### 1. Run a Code Review with `code_review`

Use `code_review` to have Devin review a pull request. Devin reviews every changed file, posts inline comments on the GitHub PR, and submits an overall review summary.

**Required parameter:**
- `pr_url` — full GitHub pull request URL (e.g. `https://github.com/org/repo/pull/123`)

**Optional parameter:**
- `instructions` — additional instructions or focus areas for the review

**What Devin reviews:**
- Code quality, readability, and maintainability
- Correctness and potential bugs
- Security concerns and vulnerabilities
- Adherence to best practices and coding conventions
- Improvement suggestions

**Example:**
```json
{
  "pr_url": "https://github.com/org/repo/pull/123",
  "instructions": "Pay close attention to SQL injection risks and input validation"
}
```

The response includes the session ID, status, pull request links, and Devin's review summary. Inline comments are posted directly on the GitHub PR. The session is archived automatically after completion.

### 2. Retrieve Results with `check_session`

If the review response is missing Devin's summary, use `check_session` with the session ID to retrieve the full results including messages and session insights.

```json
{
  "session_id": "32fee96e7997499ca010301aa50eefce"
}
```

### 3. Interpreting the Review Response

**Devin's Response section** — contains Devin's overall review summary describing what was found across the PR.

**Inline comments** — Devin posts detailed comments directly on the GitHub PR at specific lines. These are visible on the PR page in GitHub, not in the plugin response.

- If the response says "Devin returned an error in messaging", review the session directly at the provided URL
- If the response says "Devin did not return a message", the session completed but produced no message output — check the PR on GitHub for inline comments

**Session Insights** (via `check_session`) — provides additional analysis including issues found, action items, and a timeline of what Devin reviewed.

## Tips for Effective Code Reviews

- Use `instructions` to focus on specific concerns (e.g. "check for memory leaks" or "verify error propagation")
- Devin posts comments directly on GitHub, so reviewers see them alongside the diff
- Combine with `code_qa` to get both a code review and a QA test pass on the same PR
- For security-focused reviews, add instructions like "focus on authentication, authorization, and input sanitization"
