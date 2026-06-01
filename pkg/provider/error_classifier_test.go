package provider

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		wantType      string
		wantRetryable bool
		wantSwitch    bool
		wantBreaker   bool
		wantRefund    bool
	}{
		{name: "auth", statusCode: http.StatusUnauthorized, wantType: ErrorTypeAuth, wantRefund: true, wantBreaker: true},
		{name: "forbidden", statusCode: http.StatusForbidden, wantType: ErrorTypeAuth, wantRefund: true, wantBreaker: true},
		{name: "rate limit", statusCode: http.StatusTooManyRequests, wantType: ErrorTypeRateLimit, wantRetryable: true, wantSwitch: true, wantBreaker: true, wantRefund: true},
		{name: "bad request", statusCode: http.StatusBadRequest, wantType: ErrorTypeBadRequest, wantRefund: true},
		{name: "not found", statusCode: http.StatusNotFound, wantType: ErrorTypeModelNotFound, wantSwitch: true, wantBreaker: true, wantRefund: true},
		{name: "server", statusCode: http.StatusBadGateway, wantType: ErrorTypeServer, wantRetryable: true, wantSwitch: true, wantBreaker: true, wantRefund: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyHTTPStatus("openai", tt.statusCode, "upstream message")
			require.Equal(t, tt.wantType, got.Type)
			require.Equal(t, tt.statusCode, got.StatusCode)
			require.Equal(t, tt.wantRetryable, got.Retryable)
			require.Equal(t, tt.wantSwitch, got.Switchable)
			require.Equal(t, tt.wantBreaker, got.CircuitBreakerSignal)
			require.Equal(t, tt.wantRefund, got.RefundSuggested)
			require.NotContains(t, got.MessageRedacted, "sk-abcdef1234567890")
		})
	}
}

func TestClassifyErrorTimeoutAndNetwork(t *testing.T) {
	timeoutErr := &net.DNSError{IsTimeout: true}
	got := ClassifyError("openai", timeoutErr)
	require.Equal(t, ErrorTypeTimeout, got.Type)
	require.True(t, got.Retryable)
	require.True(t, got.Switchable)
	require.True(t, got.CircuitBreakerSignal)

	got = ClassifyError("openai", context.DeadlineExceeded)
	require.Equal(t, ErrorTypeTimeout, got.Type)

	got = ClassifyError("openai", errors.New("connection reset by peer"))
	require.Equal(t, ErrorTypeNetwork, got.Type)
	require.True(t, got.Retryable)
	require.True(t, got.Switchable)
}

func TestClassifyHTTPStatusDetectsContentFilter(t *testing.T) {
	got := ClassifyHTTPStatus("openai", http.StatusBadRequest, "content policy violation")

	require.Equal(t, ErrorTypeContentFilter, got.Type)
	require.False(t, got.Retryable)
	require.False(t, got.Switchable)
}

func TestClassifyErrorNil(t *testing.T) {
	require.Nil(t, ClassifyError("openai", nil))
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestClassifyErrorNetErrorTimeout(t *testing.T) {
	got := ClassifyError("openai", timeoutError{})

	require.Equal(t, ErrorTypeTimeout, got.Type)
	require.True(t, got.Retryable)
	require.True(t, got.Switchable)
	require.WithinDuration(t, time.Now(), got.ClassifiedAt, time.Second)
}
