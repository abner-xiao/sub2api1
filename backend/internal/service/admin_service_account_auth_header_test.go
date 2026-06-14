//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminService_CreateAccount_NormalizesAnthropicBearerToken(t *testing.T) {
	accountRepo := &accountRepoStub{}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: &groupRepoStub{}}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "anthropic bearer",
		Platform: PlatformAnthropic,
		Type:     AccountTypeApiKey,
		Credentials: map[string]any{
			"api_key": "token",
		},
		Extra: map[string]any{
			"auth_header": "bearer",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Len(t, accountRepo.created, 1)
	require.Equal(t, "bearer", accountRepo.created[0].Extra["anthropic_auth_header"])
	require.Equal(t, "bearer", accountRepo.created[0].Extra["auth_header"])
}

func TestAdminService_CreateAccount_LeavesOpenAIBearerTokenGeneric(t *testing.T) {
	accountRepo := &accountRepoStub{}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: &groupRepoStub{}}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "openai bearer",
		Platform: PlatformOpenAI,
		Type:     AccountTypeApiKey,
		Credentials: map[string]any{
			"api_key": "token",
		},
		Extra: map[string]any{
			"auth_header": "bearer",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Len(t, accountRepo.created, 1)
	require.Equal(t, "bearer", accountRepo.created[0].Extra["auth_header"])
	require.NotContains(t, accountRepo.created[0].Extra, "anthropic_auth_header")
}

func TestAdminService_UpdateAccount_AllowsClearingAnthropicBearerToken(t *testing.T) {
	accountRepo := &accountRepoStub{
		account: &Account{
			ID:       12,
			Name:     "anthropic bearer",
			Platform: PlatformAnthropic,
			Type:     AccountTypeApiKey,
			Extra: map[string]any{
				"anthropic_auth_header": "bearer",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: accountRepo, groupRepo: &groupRepoStub{}}

	account, err := svc.UpdateAccount(context.Background(), 12, &UpdateAccountInput{
		Extra: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Len(t, accountRepo.updated, 1)
	require.Empty(t, accountRepo.updated[0].Extra)
}
