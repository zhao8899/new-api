package dify

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestBuildsDifyChatPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello dify"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		IsStream: true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "chatflow",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	difyReq, ok := got.(*DifyChatRequest)
	require.True(t, ok)
	require.Equal(t, "streaming", difyReq.ResponseMode)
	require.Contains(t, difyReq.Query, "USER:")
	require.Contains(t, difyReq.Query, "hello dify")
	require.NotEmpty(t, difyReq.User)
}

func TestConvertClaudeRequestBuildsDifyChatPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "claude-3-5-sonnet",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{
						Type: "text",
						Text: func() *string {
							s := "hello claude"
							return &s
						}(),
					},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "chatflow",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	difyReq, ok := got.(*DifyChatRequest)
	require.True(t, ok)
	require.Equal(t, "blocking", difyReq.ResponseMode)
	require.Contains(t, difyReq.Query, "USER:")
	require.Contains(t, difyReq.Query, "hello claude")
}
