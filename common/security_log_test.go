package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeLogSecretRedactsValue(t *testing.T) {
	got := SafeLogSecret("whsec_real_secret")

	require.Contains(t, got, "redacted")
	require.Contains(t, got, "len=17")
	require.NotContains(t, got, "whsec_real_secret")
}

func TestSafeLogPayloadMasksAndFingerprintsPayload(t *testing.T) {
	payload := []byte(`{"email":"user@example.com","callback":"https://api.example.com/pay?key=secret","ip":"192.168.1.1"}`)

	got := SafeLogPayload(payload)

	require.Contains(t, got, "len=")
	require.Contains(t, got, "sha256=")
	require.NotContains(t, got, "user@example.com")
	require.NotContains(t, got, "api.example.com")
	require.NotContains(t, got, "192.168.1.1")
}

func TestSafeLogPayloadTruncatesLargePayload(t *testing.T) {
	payload := []byte(strings.Repeat("a", safeLogPayloadPreviewLimit+100))

	got := SafeLogPayload(payload)

	require.Contains(t, got, "len=612")
	require.NotContains(t, got, strings.Repeat("a", safeLogPayloadPreviewLimit+1))
}
