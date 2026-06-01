package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestCheckWebSocketOriginAllowsNonBrowserRequest(t *testing.T) {
	withWebSocketOriginPolicy(t, false, nil, "")

	req, err := http.NewRequest(http.MethodGet, "/v1/realtime", nil)
	require.NoError(t, err)

	require.True(t, checkWebSocketOrigin(req))
}

func TestCheckWebSocketOriginAllowsConfiguredOrigin(t *testing.T) {
	withWebSocketOriginPolicy(t, false, []string{"https://app.example.com"}, "")

	req, err := http.NewRequest(http.MethodGet, "/v1/realtime", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://app.example.com")

	require.True(t, checkWebSocketOrigin(req))
}

func TestCheckWebSocketOriginRejectsUnconfiguredOrigin(t *testing.T) {
	withWebSocketOriginPolicy(t, false, []string{"https://app.example.com"}, "")

	req, err := http.NewRequest(http.MethodGet, "/v1/realtime", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://evil.example.com")

	require.False(t, checkWebSocketOrigin(req))
}

func TestCheckWebSocketOriginIgnoresAllowAllInProductionSecurityMode(t *testing.T) {
	withWebSocketOriginPolicy(t, true, nil, "production")

	req, err := http.NewRequest(http.MethodGet, "/v1/realtime", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://evil.example.com")

	require.False(t, checkWebSocketOrigin(req))
}

func withWebSocketOriginPolicy(t *testing.T, allowAll bool, allowed []string, securityMode string) {
	t.Helper()

	originalAllowAll := common.CorsAllowAllOrigins
	originalAllowedOrigins := common.CorsAllowedOrigins
	common.CorsAllowAllOrigins = allowAll
	common.CorsAllowedOrigins = allowed
	t.Setenv("NEW_API_SECURITY_MODE", securityMode)
	t.Cleanup(func() {
		common.CorsAllowAllOrigins = originalAllowAll
		common.CorsAllowedOrigins = originalAllowedOrigins
	})
}
