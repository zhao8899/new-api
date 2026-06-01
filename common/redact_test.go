package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactSensitiveTextMasksSecrets(t *testing.T) {
	input := `Authorization: Bearer sk-abcdef1234567890 password=super-secret client_secret=oauth-secret bare sk-zyxw9876543210`

	got := RedactSensitiveText(input)

	require.NotContains(t, got, "sk-abcdef1234567890")
	require.NotContains(t, got, "sk-zyxw9876543210")
	require.NotContains(t, got, "super-secret")
	require.NotContains(t, got, "oauth-secret")
	require.Contains(t, got, "sk-z****3210")
	require.Contains(t, got, "****")
}

func TestRedactHeadersMasksSensitiveValues(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer sk-abcdef1234567890")
	headers.Set("Cookie", "session=secret-cookie")
	headers.Set("X-Request-ID", "req-123")

	got := RedactHeaders(headers)

	require.Equal(t, "req-123", got.Get("X-Request-ID"))
	require.NotContains(t, got.Get("Authorization"), "sk-abcdef1234567890")
	require.NotContains(t, got.Get("Cookie"), "secret-cookie")
	require.Equal(t, "Bearer ****", got.Get("Authorization"))
}

func TestRedactJSONBodyMasksSensitiveFields(t *testing.T) {
	body := []byte(`{"username":"alice","password":"super-secret","nested":{"api_key":"sk-abcdef1234567890"},"items":[{"token":"tok-secret"}]}`)

	got := string(RedactJSONBody(body))

	require.Contains(t, got, `"username":"alice"`)
	require.NotContains(t, got, "super-secret")
	require.NotContains(t, got, "sk-abcdef1234567890")
	require.NotContains(t, got, "tok-secret")
	require.Contains(t, got, `"password":"****"`)
	require.Contains(t, got, `"api_key":"****"`)
	require.Contains(t, got, `"token":"****"`)
}

func TestRedactJSONBodyFallsBackToTextRedaction(t *testing.T) {
	body := []byte(`not-json api_key=sk-abcdef1234567890`)

	got := string(RedactJSONBody(body))

	require.NotContains(t, got, "sk-abcdef1234567890")
	require.Contains(t, got, "api_key=****")
}
