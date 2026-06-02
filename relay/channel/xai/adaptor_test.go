package xai

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestSupportsSearchSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "search this"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "grok-2-search",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	payload, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "grok-2", payload["model"])

	searchParameters, ok := payload["search_parameters"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "on", searchParameters["mode"])
	require.Equal(t, "grok-2", info.UpstreamModelName)
}

func TestConvertClaudeRequestSupportsSearchSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "grok-2-search",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: xaiStringPtr("search this")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "grok-2-search",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	payload, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "grok-2", payload["model"])

	searchParameters, ok := payload["search_parameters"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "on", searchParameters["mode"])
	require.Equal(t, "grok-2", info.UpstreamModelName)
}

func xaiStringPtr(v string) *string {
	return &v
}
