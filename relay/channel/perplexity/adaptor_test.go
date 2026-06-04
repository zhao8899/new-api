package perplexity

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestAppliesPerplexityConstraints(t *testing.T) {
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
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "sonar-pro",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	pplxReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "sonar-pro", pplxReq.Model)
	require.Len(t, pplxReq.Messages, 1)
	require.Equal(t, "hello from gemini", pplxReq.Messages[0].Content)
	require.NotNil(t, pplxReq.TopP)
	require.Equal(t, 0.99, *pplxReq.TopP)
}
