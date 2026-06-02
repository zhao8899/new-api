package zhipu_4v

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestAppliesTopPClamp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	topP := 2.0
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello zhipu"},
				},
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			TopP: &topP,
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "glm-4v-plus",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	zhReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "glm-4v-plus", zhReq.Model)
	require.Len(t, zhReq.Messages, 1)
	require.Equal(t, "hello zhipu", zhReq.Messages[0].Content)
	require.NotNil(t, zhReq.TopP)
	require.Equal(t, 0.99, *zhReq.TopP)
}
