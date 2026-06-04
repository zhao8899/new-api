package moonshot

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestBuildsOpenAIRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello moonshot"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "moonshot-v1-8k",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	msReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "moonshot-v1-8k", msReq.Model)
	require.Len(t, msReq.Messages, 1)
	require.Equal(t, "user", msReq.Messages[0].Role)
	require.Equal(t, "hello moonshot", msReq.Messages[0].Content)
}
