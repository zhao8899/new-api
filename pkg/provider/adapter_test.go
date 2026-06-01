package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type testAdapter struct{}

func (testAdapter) Name() string { return "test" }
func (testAdapter) Protocol() string {
	return "openai-compatible"
}
func (testAdapter) ValidateRequest(context.Context, any) error {
	return nil
}
func (testAdapter) ConvertRequest(context.Context, any) (*http.Request, error) {
	return http.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)
}
func (testAdapter) ParseResponse(context.Context, *http.Response) (*UnifiedResponse, error) {
	return &UnifiedResponse{StatusCode: http.StatusOK}, nil
}
func (testAdapter) ParseError(context.Context, *http.Response) (*ProviderError, error) {
	return &ProviderError{Type: ErrorTypeServer, StatusCode: http.StatusInternalServerError}, nil
}

func TestAdapterContract(t *testing.T) {
	var adapter Adapter = testAdapter{}

	req, err := adapter.ConvertRequest(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, "test", adapter.Name())
	require.Equal(t, "openai-compatible", adapter.Protocol())
	require.Equal(t, "https://example.com/v1/chat/completions", req.URL.String())
}
