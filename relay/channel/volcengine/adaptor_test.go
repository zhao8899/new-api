package volcengine

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
					{Text: "hello volcengine"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "doubao-pro-32k",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	veReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "doubao-pro-32k", veReq.Model)
	require.Len(t, veReq.Messages, 1)
	require.Equal(t, "hello volcengine", veReq.Messages[0].Content)
}
