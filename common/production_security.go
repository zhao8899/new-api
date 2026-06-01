package common

import (
	"crypto/subtle"
	"fmt"
	"os"
	"strings"
)

func IsProductionSecurityMode() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("NEW_API_SECURITY_MODE")))
	return value == "production" || value == "prod" || value == "financial"
}

func IsProductionRuntime() bool {
	if IsProductionSecurityMode() {
		return true
	}
	for _, key := range []string{"NEW_API_ENV", "APP_ENV", "ENV", "NODE_ENV", "GIN_MODE"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "production" || value == "prod" || value == "release" {
			return true
		}
	}
	return false
}

func ValidateProductionSecurityConfig(sessionSecretSet, cryptoSecretSet bool) error {
	if !IsProductionRuntime() {
		return nil
	}
	if !sessionSecretSet {
		return fmt.Errorf("SESSION_SECRET must be set in production")
	}
	if !cryptoSecretSet {
		return fmt.Errorf("CRYPTO_SECRET must be set in production")
	}
	if !SessionCookieSecure {
		return fmt.Errorf("SESSION_COOKIE_SECURE=true is required in production")
	}
	if TLSInsecureSkipVerify {
		return fmt.Errorf("TLS_INSECURE_SKIP_VERIFY=true is not allowed in production")
	}
	if IsProductionSecurityMode() {
		if strings.TrimSpace(os.Getenv("SQL_DSN")) == "" {
			return fmt.Errorf("SQL_DSN must be set when NEW_API_SECURITY_MODE=production")
		}
		if strings.TrimSpace(os.Getenv("REDIS_CONN_STRING")) == "" {
			return fmt.Errorf("REDIS_CONN_STRING must be set when NEW_API_SECURITY_MODE=production")
		}
	}
	return nil
}

func SetupTokenRequired() bool {
	return strings.TrimSpace(os.Getenv("NEW_API_SETUP_TOKEN")) != "" || IsProductionSecurityMode()
}

func ValidateSetupToken(token string) error {
	expected := strings.TrimSpace(os.Getenv("NEW_API_SETUP_TOKEN"))
	if !SetupTokenRequired() {
		return nil
	}
	if expected == "" {
		return fmt.Errorf("setup token is required but NEW_API_SETUP_TOKEN is not configured")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(token)), []byte(expected)) != 1 {
		return fmt.Errorf("invalid setup token")
	}
	return nil
}
