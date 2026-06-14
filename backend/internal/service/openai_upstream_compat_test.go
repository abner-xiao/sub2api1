//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveOpenAIUpstreamURL(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		mode    openAIUpstreamMode
		want    string
	}{
		{
			name: "official base adds v1 responses",
			account: &Account{
				Platform:    PlatformOpenAI,
				Type:        AccountTypeApiKey,
				Credentials: map[string]any{"base_url": "https://api.openai.com"},
			},
			mode: openAIUpstreamModeResponses,
			want: "https://api.openai.com/v1/responses",
		},
		{
			name: "v1 base adds responses",
			account: &Account{
				Platform:    PlatformOpenAI,
				Type:        AccountTypeApiKey,
				Credentials: map[string]any{"base_url": "https://api.openai.com/v1"},
			},
			mode: openAIUpstreamModeResponses,
			want: "https://api.openai.com/v1/responses",
		},
		{
			name: "sensenova v1 uses chat completions",
			account: &Account{
				Platform:    PlatformOpenAI,
				Type:        AccountTypeApiKey,
				Credentials: map[string]any{"base_url": "https://token.sensenova.cn/v1"},
			},
			mode: openAIUpstreamModeChatCompletions,
			want: "https://token.sensenova.cn/v1/chat/completions",
		},
		{
			name: "full chat completions endpoint preserved",
			account: &Account{
				Platform:    PlatformOpenAI,
				Type:        AccountTypeApiKey,
				Credentials: map[string]any{"base_url": "https://example.com/v1/chat/completions"},
			},
			mode: openAIUpstreamModeChatCompletions,
			want: "https://example.com/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, resolveOpenAIUpstreamURL(tt.account, tt.mode))
		})
	}
}

func TestResolveOpenAIUpstreamMode(t *testing.T) {
	require.Equal(t, openAIUpstreamModeChatCompletions, resolveOpenAIUpstreamMode(&Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeApiKey,
		Credentials: map[string]any{"base_url": "https://token.sensenova.cn/v1"},
	}))
	require.Equal(t, openAIUpstreamModeChatCompletions, resolveOpenAIUpstreamMode(&Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeApiKey,
		Credentials: map[string]any{"base_url": "https://example.com/v1/chat/completions"},
	}))
	require.Equal(t, openAIUpstreamModeResponses, resolveOpenAIUpstreamMode(&Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeApiKey,
		Credentials: map[string]any{"base_url": "https://api.openai.com/v1/responses"},
	}))
}

func TestConvertResponsesRequestToChatCompletions(t *testing.T) {
	chatBody, err := convertResponsesRequestToChatCompletions(map[string]any{
		"model":        "SenseChat-5",
		"instructions": "system prompt",
		"input": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "hello"},
				},
			},
		},
		"stream":            true,
		"max_output_tokens": 128,
	})

	require.NoError(t, err)
	require.Equal(t, "SenseChat-5", chatBody["model"])
	require.Equal(t, true, chatBody["stream"])
	require.Equal(t, 128, chatBody["max_tokens"])
	messages, ok := chatBody["messages"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, messages, 2)
	require.Equal(t, "system", messages[0]["role"])
	require.Equal(t, "system prompt", messages[0]["content"])
	require.Equal(t, "user", messages[1]["role"])
	require.Equal(t, "hello", messages[1]["content"])
}
