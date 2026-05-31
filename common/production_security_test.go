package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateProductionSecurityConfigNoopsOutsideProduction(t *testing.T) {
	t.Setenv("NEW_API_ENV", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("ENV", "")
	t.Setenv("NODE_ENV", "")
	t.Setenv("GIN_MODE", "")

	originalSecure := SessionCookieSecure
	originalTLS := TLSInsecureSkipVerify
	SessionCookieSecure = false
	TLSInsecureSkipVerify = true
	t.Cleanup(func() {
		SessionCookieSecure = originalSecure
		TLSInsecureSkipVerify = originalTLS
	})

	require.NoError(t, ValidateProductionSecurityConfig(false, false))
}

func TestValidateProductionSecurityConfigRequiresProductionDefaults(t *testing.T) {
	t.Setenv("NEW_API_ENV", "production")

	originalSecure := SessionCookieSecure
	originalTLS := TLSInsecureSkipVerify
	t.Cleanup(func() {
		SessionCookieSecure = originalSecure
		TLSInsecureSkipVerify = originalTLS
	})

	SessionCookieSecure = false
	TLSInsecureSkipVerify = false
	require.ErrorContains(t, ValidateProductionSecurityConfig(true, true), "SESSION_COOKIE_SECURE")

	SessionCookieSecure = true
	TLSInsecureSkipVerify = true
	require.ErrorContains(t, ValidateProductionSecurityConfig(true, true), "TLS_INSECURE_SKIP_VERIFY")

	TLSInsecureSkipVerify = false
	require.ErrorContains(t, ValidateProductionSecurityConfig(false, true), "SESSION_SECRET")
	require.ErrorContains(t, ValidateProductionSecurityConfig(true, false), "CRYPTO_SECRET")
	require.NoError(t, ValidateProductionSecurityConfig(true, true))
}
