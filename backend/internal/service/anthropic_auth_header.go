package service

import (
	"net/http"
	"strings"
)

const (
	AnthropicAuthHeaderBearer = "bearer"
	OpenAIAuthHeaderBearer    = "bearer"

	anthropicAuthHeaderExtraKey = "anthropic_auth_header"
	openAIAuthHeaderExtraKey    = "openai_auth_header"
	accountAuthHeaderExtraKey   = "auth_header"
)

// UseAnthropicBearerAuth reports whether an Anthropic API Key account should
// send Authorization: Bearer instead of the official x-api-key header.
func (a *Account) UseAnthropicBearerAuth() bool {
	if a == nil || a.Platform != PlatformAnthropic || a.Type != AccountTypeApiKey || a.Extra == nil {
		return false
	}
	mode := a.GetExtraString(anthropicAuthHeaderExtraKey)
	if mode == "" {
		mode = a.GetExtraString(accountAuthHeaderExtraKey)
	}
	return strings.EqualFold(strings.TrimSpace(mode), AnthropicAuthHeaderBearer)
}

// UseOpenAIBearerAuth reports whether an OpenAI API Key account should send
// Authorization: Bearer instead of API-key style headers.
func (a *Account) UseOpenAIBearerAuth() bool {
	if a == nil || a.Platform != PlatformOpenAI || a.Type != AccountTypeApiKey || a.Extra == nil {
		return false
	}
	mode := a.GetExtraString(openAIAuthHeaderExtraKey)
	if mode == "" {
		mode = a.GetExtraString(accountAuthHeaderExtraKey)
	}
	return strings.EqualFold(strings.TrimSpace(mode), OpenAIAuthHeaderBearer)
}

func normalizeAccountAuthHeader(platform, accountType string, extra map[string]any) map[string]any {
	if accountType != AccountTypeApiKey || extra == nil {
		return extra
	}

	mode, _ := extra[accountAuthHeaderExtraKey].(string)
	if !strings.EqualFold(strings.TrimSpace(mode), AnthropicAuthHeaderBearer) {
		return extra
	}

	normalized := make(map[string]any, len(extra)+1)
	for k, v := range extra {
		normalized[k] = v
	}
	switch platform {
	case PlatformAnthropic:
		normalized[anthropicAuthHeaderExtraKey] = AnthropicAuthHeaderBearer
	case PlatformOpenAI:
		normalized[openAIAuthHeaderExtraKey] = OpenAIAuthHeaderBearer
	}
	return normalized
}

func setAnthropicAPIKeyAuthHeader(h http.Header, account *Account, token string) {
	if account.UseAnthropicBearerAuth() {
		deleteAnthropicAuthHeaders(h)
		h.Set("Authorization", "Bearer "+token)
		return
	}

	deleteAnthropicAuthHeaders(h)
	h.Set("x-api-key", token)
}

func setOpenAIAPIKeyAuthHeader(h http.Header, account *Account, token string) {
	if account.UseOpenAIBearerAuth() {
		deleteOpenAIAuthHeaders(h)
		h.Set("Authorization", "Bearer "+token)
		return
	}

	deleteOpenAIAuthHeaders(h)
	h.Set("x-api-key", token)
	h.Set("api-key", token)
}

func deleteAnthropicAuthHeaders(h http.Header) {
	h.Del("Authorization")
	h.Del("authorization")
	h.Del("x-api-key")
	h.Del("X-Api-Key")
	delete(h, "authorization")
	delete(h, "x-api-key")
	delete(h, "X-Api-Key")
}

func deleteOpenAIAuthHeaders(h http.Header) {
	h.Del("Authorization")
	h.Del("authorization")
	h.Del("x-api-key")
	h.Del("X-Api-Key")
	h.Del("api-key")
	h.Del("Api-Key")
	delete(h, "authorization")
	delete(h, "x-api-key")
	delete(h, "X-Api-Key")
	delete(h, "api-key")
	delete(h, "Api-Key")
}
