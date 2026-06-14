package service

import (
	"net/http"
	"testing"
)

func TestSetAnthropicAPIKeyAuthHeader(t *testing.T) {
	t.Run("default uses x-api-key", func(t *testing.T) {
		header := http.Header{}
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeApiKey,
		}

		setAnthropicAPIKeyAuthHeader(header, account, "api-key")

		if got := header.Get("x-api-key"); got != "api-key" {
			t.Fatalf("x-api-key = %q, want api-key", got)
		}
		if got := header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
	})

	t.Run("bearer uses Authorization", func(t *testing.T) {
		header := http.Header{"x-api-key": []string{"inbound"}}
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeApiKey,
			Extra: map[string]any{
				"anthropic_auth_header": "bearer",
			},
		}

		setAnthropicAPIKeyAuthHeader(header, account, "api-key")

		if got := header.Get("Authorization"); got != "Bearer api-key" {
			t.Fatalf("Authorization = %q, want Bearer api-key", got)
		}
		if got := header.Get("x-api-key"); got != "" {
			t.Fatalf("x-api-key = %q, want empty", got)
		}
	})

	t.Run("generic auth header bearer uses Authorization", func(t *testing.T) {
		header := http.Header{"x-api-key": []string{"inbound"}}
		account := &Account{
			Platform: PlatformAnthropic,
			Type:     AccountTypeApiKey,
			Extra: map[string]any{
				"auth_header": "Bearer",
			},
		}

		setAnthropicAPIKeyAuthHeader(header, account, "api-key")

		if got := header.Get("Authorization"); got != "Bearer api-key" {
			t.Fatalf("Authorization = %q, want Bearer api-key", got)
		}
		if got := header.Get("x-api-key"); got != "" {
			t.Fatalf("x-api-key = %q, want empty", got)
		}
	})

	t.Run("generic auth header is ignored for openai account", func(t *testing.T) {
		header := http.Header{}
		account := &Account{
			Platform: PlatformOpenAI,
			Type:     AccountTypeApiKey,
			Extra: map[string]any{
				"auth_header": "bearer",
			},
		}

		setAnthropicAPIKeyAuthHeader(header, account, "api-key")

		if got := header.Get("x-api-key"); got != "api-key" {
			t.Fatalf("x-api-key = %q, want api-key", got)
		}
		if got := header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
	})
}
