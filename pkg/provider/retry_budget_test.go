package provider

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryBudgetAllowsOneServerErrorFallbackByDefault(t *testing.T) {
	budget := NewRetryBudget(RetryBudgetOptions{})
	providerErr := ClassifyHTTPStatus("openai", http.StatusBadGateway, "upstream unavailable")

	decision := budget.Decide(providerErr)

	require.True(t, decision.Retry)
	require.True(t, decision.SwitchChannel)
	require.Equal(t, 1, decision.NextRetryCount)
	require.Equal(t, "SERVER_ERROR", decision.ErrorType)
}

func TestRetryBudgetRejectsAuthAndBadRequest(t *testing.T) {
	budget := NewRetryBudget(RetryBudgetOptions{})

	authDecision := budget.Decide(ClassifyHTTPStatus("openai", http.StatusUnauthorized, "invalid key"))
	badRequestDecision := budget.Decide(ClassifyHTTPStatus("openai", http.StatusBadRequest, "invalid request"))

	require.False(t, authDecision.Retry)
	require.False(t, authDecision.SwitchChannel)
	require.Equal(t, "auth errors are not retryable", authDecision.Reason)
	require.False(t, badRequestDecision.Retry)
	require.Equal(t, "bad request errors are not retryable", badRequestDecision.Reason)
}

func TestRetryBudgetNeverRetriesAfterStreamStarted(t *testing.T) {
	budget := NewRetryBudget(RetryBudgetOptions{
		StreamStarted: true,
	})
	providerErr := ClassifyHTTPStatus("openai", http.StatusInternalServerError, "partial stream failure")

	decision := budget.Decide(providerErr)

	require.False(t, decision.Retry)
	require.Equal(t, "stream already produced output", decision.Reason)
}

func TestRetryBudgetNeverRetriesNonIdempotentTask(t *testing.T) {
	budget := NewRetryBudget(RetryBudgetOptions{
		NonIdempotent: true,
	})
	providerErr := ClassifyHTTPStatus("suno", http.StatusInternalServerError, "task creation failed")

	decision := budget.Decide(providerErr)

	require.False(t, decision.Retry)
	require.Equal(t, "request is not idempotent", decision.Reason)
}

func TestRetryBudgetStopsAtConfiguredMaxRetries(t *testing.T) {
	budget := NewRetryBudget(RetryBudgetOptions{
		RetryCount: 1,
		MaxRetries: 1,
	})
	providerErr := ClassifyHTTPStatus("gemini", http.StatusTooManyRequests, "quota exceeded")

	decision := budget.Decide(providerErr)

	require.False(t, decision.Retry)
	require.Equal(t, "retry budget exhausted", decision.Reason)
}

func TestRetryBudgetDoesNotRetryRateLimitOnSameChannel(t *testing.T) {
	budget := NewRetryBudget(RetryBudgetOptions{})
	providerErr := ClassifyHTTPStatus("gemini", http.StatusTooManyRequests, "quota exceeded")

	decision := budget.Decide(providerErr)

	require.True(t, decision.Retry)
	require.True(t, decision.SwitchChannel)
	require.False(t, decision.SameChannel)
	require.Equal(t, 1, decision.NextRetryCount)
}
