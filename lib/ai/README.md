# AI Package

A Go package that provides a simplified wrapper around the `go-openai` library, designed to make AI integration easier and more convenient.

## Features

- Simple wrapper around `go-openai` client
- Pre-configured providers (OpenAI, OpenRouter)
- Convenience methods for common Git-related AI tasks
- Easy configuration and setup

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/input-output-hk/catalyst-forge/lib/ai"
)

func main() {
    // Using OpenRouter
    config := ai.OpenRouterConfig("your-api-key", "openai/gpt-4o-mini")
    client := ai.NewClient(config)

    // Simple chat completion
    response, err := client.ChatCompletion(context.Background(), "Hello, world!")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response)
}
```

### Git-specific Functions

```go
// Summarize a git diff
summary, err := client.SummarizeDiff(ctx, diffString)

// Generate a commit message from a diff
commitMsg, err := client.GenerateCommitMessage(ctx, diffString)

// Analyze code
analysis, err := client.AnalyzeCode(ctx, codeString, "security vulnerabilities")
```

### Configuration Options

```go
// OpenRouter configuration
config := ai.OpenRouterConfig("api-key", "openai/gpt-4o-mini")

// OpenAI configuration  
config := ai.OpenAIConfig("api-key", "gpt-4")

// Custom configuration
config := ai.ClientConfig{
    APIKey:  "your-key",
    BaseURL: "https://custom-api.com/v1",
    Model:   "custom-model",
}
```

## API Reference

### Client

- `NewClient(config ClientConfig) *Client` - Create a new AI client
- `ChatCompletion(ctx context.Context, message string) (string, error)` - Send a chat completion request
- `SetModel(model string)` - Change the model for subsequent requests
- `GetModel() string` - Get the current model

### Git Functions

- `SummarizeDiff(ctx context.Context, diff string) (string, error)` - Analyze and summarize a git diff
- `GenerateCommitMessage(ctx context.Context, diff string) (string, error)` - Generate a commit message from a diff
- `AnalyzeCode(ctx context.Context, code string, analysisType string) (string, error)` - Analyze code for specific concerns

### Provider Helpers

- `OpenRouterConfig(apiKey, model string) ClientConfig` - Create OpenRouter configuration
- `OpenAIConfig(apiKey, model string) ClientConfig` - Create OpenAI configuration