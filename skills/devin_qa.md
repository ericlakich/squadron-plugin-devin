# Devin QA Review Skill Guide

Use the Devin plugin to perform automated QA reviews of pull requests.

## Workflow

### 1. Run a QA Review with `code_qa`

Use `code_qa` to have Devin perform a full QA review of a pull request. Devin checks out the PR branch, analyzes the changes, runs existing tests, and returns a comprehensive summary.

**Required parameter:**
- `pr_url` — full GitHub pull request URL (e.g. `https://github.com/org/repo/pull/123`)

**Optional parameter:**
- `instructions` — additional instructions or focus areas for the QA review

**What Devin checks:**
- Bug detection, edge cases, and logic errors
- Error handling adequacy
- Test execution and failure reporting
- Missing test coverage for new or changed code
- Regression risks in related functionality
- Alignment with PR description and linked issues
- Performance concerns

**Example:**
```json
{
  "pr_url": "https://github.com/org/repo/pull/123",
  "instructions": "Focus on the new payment processing logic and verify edge cases around refunds"
}
```

The response includes the session ID, status, and Devin's QA findings. The session is archived automatically after completion.

### 2. Retrieve Results with `check_session`

If the QA response is missing Devin's findings, use `check_session` with the session ID to retrieve the full results including messages and session insights.

```json
{
  "session_id": "32fee96e7997499ca010301aa50eefce"
}
```

### 3. Interpreting the QA Response

**Devin's Response section** — contains Devin's QA summary with categorized findings (critical issues, warnings, suggestions, and things that look good).

- If the response says "Devin returned an error in messaging", review the session directly at the provided URL
- If the response says "Devin did not return a message", the session completed but produced no message output — continue to the next task

**Session Insights** (via `check_session`) — provides additional analysis including issues found, action items, and a timeline of what Devin reviewed.

## Tips for Effective QA Reviews

- Use `instructions` to direct Devin toward areas of concern (e.g. "focus on concurrency safety" or "verify database migration rollback")
- For large PRs, narrow the scope with instructions like "focus on changes in the auth module"
- Combine with `code_review` for both QA testing and code-level review on the same PR
