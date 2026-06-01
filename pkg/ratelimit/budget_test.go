package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReserveAllowsWithinRPMAndTPM(t *testing.T) {
	store := newMemoryStore(time.Minute)
	limiter := NewLimiter(store)

	result, err := limiter.Reserve(context.Background(), Reservation{
		Provider:        "openai",
		Model:           "gpt-4.1",
		ChannelID:       100,
		Scope:           ScopeToken,
		ScopeKey:        "tok-1",
		EstimatedTokens: 300,
	}, Limits{RPM: 2, TPM: 1000, Window: time.Minute})

	require.NoError(t, err)
	require.True(t, result.Allowed)
	require.Equal(t, int64(1), result.RPMUsed)
	require.Equal(t, int64(300), result.TPMUsed)
	require.Contains(t, result.RPMKey, "rate_limit:openai:gpt-4.1:100:token:tok-1:rpm")
}

func TestReserveRejectsAndRollsBackRPMWhenLimitExceeded(t *testing.T) {
	store := newMemoryStore(time.Minute)
	limiter := NewLimiter(store)
	ctx := context.Background()
	reservation := Reservation{
		Provider:        "gemini",
		Model:           "gemini-2.5-pro",
		ChannelID:       200,
		Scope:           ScopeUser,
		ScopeKey:        "42",
		EstimatedTokens: 100,
	}

	first, err := limiter.Reserve(ctx, reservation, Limits{RPM: 1, TPM: 1000, Window: time.Minute})
	require.NoError(t, err)
	require.True(t, first.Allowed)

	second, err := limiter.Reserve(ctx, reservation, Limits{RPM: 1, TPM: 1000, Window: time.Minute})
	require.NoError(t, err)
	require.False(t, second.Allowed)
	require.Equal(t, LimitReasonRPM, second.Reason)
	require.Greater(t, second.RetryAfter, time.Duration(0))

	require.Equal(t, int64(1), store.value(second.RPMKey))
	require.Equal(t, int64(100), store.value(second.TPMKey))
}

func TestReserveRejectsAndRollsBackTPMWhenLimitExceeded(t *testing.T) {
	store := newMemoryStore(time.Minute)
	limiter := NewLimiter(store)

	result, err := limiter.Reserve(context.Background(), Reservation{
		Provider:        "anthropic",
		Model:           "claude-sonnet",
		ChannelID:       300,
		Scope:           ScopeGroup,
		ScopeKey:        "vip",
		EstimatedTokens: 1200,
	}, Limits{RPM: 10, TPM: 1000, Window: time.Minute})

	require.NoError(t, err)
	require.False(t, result.Allowed)
	require.Equal(t, LimitReasonTPM, result.Reason)
	require.Equal(t, int64(0), store.value(result.RPMKey))
	require.Equal(t, int64(0), store.value(result.TPMKey))
}

func TestReleaseReservationSubtractsPreReservedCapacity(t *testing.T) {
	store := newMemoryStore(time.Minute)
	limiter := NewLimiter(store)
	ctx := context.Background()
	reservation := Reservation{
		Provider:        "deepseek",
		Model:           "deepseek-chat",
		ChannelID:       400,
		Scope:           ScopeModel,
		ScopeKey:        "deepseek-chat",
		EstimatedTokens: 500,
	}

	result, err := limiter.Reserve(ctx, reservation, Limits{RPM: 10, TPM: 1000, Window: time.Minute})
	require.NoError(t, err)
	require.True(t, result.Allowed)

	err = limiter.Release(ctx, result, 300)
	require.NoError(t, err)

	require.Equal(t, int64(0), store.value(result.RPMKey))
	require.Equal(t, int64(300), store.value(result.TPMKey))
}
