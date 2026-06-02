package aws

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestBuildsClaudePayloadForBedrock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello aws"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "anthropic.claude-3-5-sonnet-20240620-v1:0",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	claudeReq, ok := got.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.Len(t, claudeReq.Messages, 1)
	require.Equal(t, "user", claudeReq.Messages[0].Role)
	require.Equal(t, "hello aws", claudeReq.Messages[0].GetStringContent())
}
