package ai

// Common provider configurations for easier setup

// OpenRouterConfig creates a ClientConfig for OpenRouter
func OpenRouterConfig(apiKey, model string) ClientConfig {
	if model == "" {
		model = "openai/gpt-4o-mini" // Default OpenRouter model
	}
	
	return ClientConfig{
		APIKey:  apiKey,
		BaseURL: "https://openrouter.ai/api/v1",
		Model:   model,
	}
}

// OpenAIConfig creates a ClientConfig for OpenAI
func OpenAIConfig(apiKey, model string) ClientConfig {
	if model == "" {
		model = "gpt-3.5-turbo" // Default OpenAI model
	}
	
	return ClientConfig{
		APIKey: apiKey,
		Model:  model,
		// BaseURL defaults to OpenAI's API
	}
}