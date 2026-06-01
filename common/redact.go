package common

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var sensitiveFieldNames = map[string]struct{}{
	"authorization":  {},
	"cookie":         {},
	"set-cookie":     {},
	"password":       {},
	"passwd":         {},
	"key":            {},
	"secret":         {},
	"token":          {},
	"access_token":   {},
	"refresh_token":  {},
	"api_key":        {},
	"apikey":         {},
	"client_secret":  {},
	"channel_key":    {},
	"stripe_api_key": {},
}

var (
	bearerSecretPattern = regexp.MustCompile(`(?i)\b(Authorization\s*:\s*Bearer\s+)[^\s,;]+`)
	keyValuePattern     = regexp.MustCompile(`(?i)\b(password|passwd|secret|token|access_token|refresh_token|api_key|apikey|client_secret|channel_key|key)\s*[:=]\s*("[^"]+"|'[^']+'|[^\s,;]+)`)
	skSecretPattern     = regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{8,})\b`)
)

func RedactSensitiveText(input string) string {
	if input == "" {
		return input
	}
	redacted := bearerSecretPattern.ReplaceAllString(input, "${1}****")
	redacted = keyValuePattern.ReplaceAllStringFunc(redacted, func(match string) string {
		parts := regexp.MustCompile(`[:=]`).Split(match, 2)
		if len(parts) != 2 {
			return "****"
		}
		separator := "="
		if strings.Contains(match, ":") {
			separator = ":"
		}
		return strings.TrimSpace(parts[0]) + separator + "****"
	})
	redacted = skSecretPattern.ReplaceAllStringFunc(redacted, maskSecretValue)
	return redacted
}

func RedactHeaders(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	redacted := make(http.Header, len(headers))
	for key, values := range headers {
		copied := make([]string, 0, len(values))
		if isSensitiveFieldName(key) {
			for _, value := range values {
				copied = append(copied, redactHeaderValue(key, value))
			}
		} else {
			copied = append(copied, values...)
		}
		redacted[key] = copied
	}
	return redacted
}

func RedactJSONBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var value any
	if err := Unmarshal(body, &value); err != nil {
		return []byte(RedactSensitiveText(string(body)))
	}
	redacted := redactJSONValue(value, "")
	data, err := Marshal(redacted)
	if err != nil {
		return []byte(RedactSensitiveText(string(body)))
	}
	return data
}

func isSensitiveFieldName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if _, ok := sensitiveFieldNames[normalized]; ok {
		return true
	}
	return strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "apikey")
}

func redactHeaderValue(key string, value string) string {
	if strings.EqualFold(key, "Authorization") {
		parts := strings.Fields(value)
		if len(parts) > 0 {
			return parts[0] + " ****"
		}
	}
	return "****"
}

func redactJSONValue(value any, key string) any {
	if isSensitiveFieldName(key) {
		return "****"
	}
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			redacted[childKey] = redactJSONValue(childValue, childKey)
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, childValue := range typed {
			redacted[index] = redactJSONValue(childValue, key)
		}
		return redacted
	case string:
		return RedactSensitiveText(typed)
	default:
		return typed
	}
}

func maskSecretValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	if strings.HasPrefix(value, "sk-") && len(value) > 10 {
		return fmt.Sprintf("%s****%s", value[:4], value[len(value)-4:])
	}
	return fmt.Sprintf("%s****%s", value[:2], value[len(value)-2:])
}
