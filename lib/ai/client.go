package ai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// Client wraps the go-openai client to provide easier usage patterns
type Client struct {
	client *openai.Client
	model  string
}

// ClientConfig holds configuration for creating a new AI client
type ClientConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewClient creates a new AI client with the given configuration
func NewClient(config ClientConfig) *Client {
	openaiConfig := openai.DefaultConfig(config.APIKey)
	
	if config.BaseURL != "" {
		openaiConfig.BaseURL = config.BaseURL
	}
	
	model := config.Model
	if model == "" {
		model = "gpt-3.5-turbo" // Default model
	}
	
	client := openai.NewClientWithConfig(openaiConfig)
	
	return &Client{
		client: client,
		model:  model,
	}
}

// ChatCompletion sends a simple chat completion request with a single user message
func (c *Client) ChatCompletion(ctx context.Context, message string) (string, error) {
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: message,
				},
			},
		},
	)
	
	if err != nil {
		return "", fmt.Errorf("chat completion error: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}
	
	return resp.Choices[0].Message.Content, nil
}

// SetModel allows changing the model for subsequent requests
func (c *Client) SetModel(model string) {
	c.model = model
}

// GetModel returns the current model being used
func (c *Client) GetModel() string {
	return c.model
}