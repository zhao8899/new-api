package deepseek

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestAppliesDeepSeekThinkingSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello deepseek"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "deepseek-v4-chat-max",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	dsReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "deepseek-v4-chat", dsReq.Model)
	require.Equal(t, "deepseek-v4-chat", info.UpstreamModelName)
	require.Equal(t, "max", dsReq.ReasoningEffort)
	require.Equal(t, "max", info.ReasoningEffort)
	require.JSONEq(t, `{"type":"enabled"}`, string(dsReq.THINKING))
	require.Len(t, dsReq.Messages, 1)
	require.Equal(t, "hello deepseek", dsReq.Messages[0].Content)
}
