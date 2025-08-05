package ai

import (
	"context"
	"fmt"
)

// SummarizeDiff analyzes a git diff and provides a structured summary
func (c *Client) SummarizeDiff(ctx context.Context, diff string) (string, error) {
	prompt := fmt.Sprintf(`Please analyze the following Git diff and provide a concise summary of the changes made:

%s

Please provide:
1. A brief summary of what changed
2. Which files were modified, added, or deleted
3. The general purpose or intent of these changes

Keep the summary clear and concise.`, diff)

	return c.ChatCompletion(ctx, prompt)
}

// AnalyzeCode analyzes code and provides insights
func (c *Client) AnalyzeCode(ctx context.Context, code string, analysisType string) (string, error) {
	prompt := fmt.Sprintf(`Please analyze the following code for %s:

%s

Provide a clear and actionable analysis.`, analysisType, code)

	return c.ChatCompletion(ctx, prompt)
}

// GenerateCommitMessage generates a commit message based on a diff
func (c *Client) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := fmt.Sprintf(`Based on the following Git diff, generate a concise commit message following conventional commit format:

%s

Format: <type>(<scope>): <description>

Where type is one of: feat, fix, docs, style, refactor, test, chore
Keep the description under 50 characters and be specific about what changed.`, diff)

	return c.ChatCompletion(ctx, prompt)
}
