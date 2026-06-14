package service

import (
	"net/http"
	"strings"
)

const (
	AnthropicAuthHeaderBearer = "bearer"

	anthropicAuthHeaderExtraKey = "anthropic_auth_header"
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

func normalizeAccountAuthHeader(platform, accountType string, extra map[string]any) map[string]any {
	if platform != PlatformAnthropic || accountType != AccountTypeApiKey || extra == nil {
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
	normalized[anthropicAuthHeaderExtraKey] = AnthropicAuthHeaderBearer
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

func deleteAnthropicAuthHeaders(h http.Header) {
	h.Del("Authorization")
	h.Del("authorization")
	h.Del("x-api-key")
	h.Del("X-Api-Key")
	delete(h, "authorization")
	delete(h, "x-api-key")
	delete(h, "X-Api-Key")
}
