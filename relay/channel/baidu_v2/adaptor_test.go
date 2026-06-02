package baidu_v2

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
					{Text: "hello baidu"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "ernie-4.5-search",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	payload, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ernie-4.5", payload["model"])
	require.Equal(t, "ernie-4.5", info.UpstreamModelName)

	webSearch, ok := payload["web_search"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, webSearch["enable"])
	require.Equal(t, true, webSearch["enable_citation"])
}
