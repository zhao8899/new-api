package cloudflare

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestReturnsOpenAIRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "@cf/meta/llama-3.1-8b-instruct",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: cloudflareStringPtr("hello cloudflare")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeChatCompletions,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "@cf/meta/llama-3.1-8b-instruct",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	openaiReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "@cf/meta/llama-3.1-8b-instruct", openaiReq.Model)
	require.Len(t, openaiReq.Messages, 1)
	content := openaiReq.Messages[0].ParseContent()
	require.Len(t, content, 1)
	require.Equal(t, "hello cloudflare", content[0].Text)
}

func cloudflareStringPtr(v string) *string {
	return &v
}
