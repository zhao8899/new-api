package common

import (
	"fmt"
	"os"
	"strings"
)

func IsProductionRuntime() bool {
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
	return nil
}
