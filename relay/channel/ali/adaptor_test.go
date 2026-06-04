package ali

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestBridgesThroughOpenAIConversion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	topP := 2.0
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello from gemini"},
				},
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			TopP: &topP,
		},
	}
	info := &relaycommon.RelayInfo{
		IsStream: false,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "qwen-plus",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	aliReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "qwen-plus", aliReq.Model)
	require.Len(t, aliReq.Messages, 1)
	require.Equal(t, "user", aliReq.Messages[0].Role)
	require.Equal(t, "hello from gemini", aliReq.Messages[0].Content)
	require.NotNil(t, aliReq.TopP)
	require.Equal(t, 0.999, *aliReq.TopP)
}
