package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default model", func(t *testing.T) {
		config := ClientConfig{
			APIKey: "test-key",
		}

		client := NewClient(config)

		assert.NotNil(t, client)
		assert.Equal(t, "gpt-3.5-turbo", client.GetModel())
	})

	t.Run("creates client with custom model", func(t *testing.T) {
		config := ClientConfig{
			APIKey: "test-key",
			Model:  "gpt-4",
		}

		client := NewClient(config)

		assert.NotNil(t, client)
		assert.Equal(t, "gpt-4", client.GetModel())
	})

	t.Run("creates client with custom base URL", func(t *testing.T) {
		config := ClientConfig{
			APIKey:  "test-key",
			BaseURL: "https://custom.api.com/v1",
			Model:   "custom-model",
		}

		client := NewClient(config)

		assert.NotNil(t, client)
		assert.Equal(t, "custom-model", client.GetModel())
	})
}

func TestSetModel(t *testing.T) {
	config := ClientConfig{
		APIKey: "test-key",
	}

	client := NewClient(config)
	assert.Equal(t, "gpt-3.5-turbo", client.GetModel())

	client.SetModel("gpt-4")
	assert.Equal(t, "gpt-4", client.GetModel())
}

func TestOpenRouterConfig(t *testing.T) {
	config := OpenRouterConfig("test-key", "")

	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, "https://openrouter.ai/api/v1", config.BaseURL)
	assert.Equal(t, "openai/gpt-4o-mini", config.Model)
}

func TestOpenAIConfig(t *testing.T) {
	config := OpenAIConfig("test-key", "")

	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, "", config.BaseURL) // Default OpenAI URL
	assert.Equal(t, "gpt-3.5-turbo", config.Model)
}
