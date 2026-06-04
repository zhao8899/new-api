package mistral

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
					{Text: "hello mistral"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "mistral-large-latest",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	msReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "mistral-large-latest", msReq.Model)
	require.Len(t, msReq.Messages, 1)
	content, ok := msReq.Messages[0].Content.([]dto.MediaContent)
	require.True(t, ok)
	require.Len(t, content, 1)
	require.Equal(t, "text", content[0].Type)
	require.Equal(t, "hello mistral", content[0].Text)
}

func TestConvertClaudeRequestBuildsMistralRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "mistral-large-latest",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: mistralStringPtr("hello mistral")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "mistral-large-latest",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	msReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Equal(t, "mistral-large-latest", msReq.Model)
	require.Len(t, msReq.Messages, 1)
	content, ok := msReq.Messages[0].Content.([]dto.MediaContent)
	require.True(t, ok)
	require.Len(t, content, 1)
	require.Equal(t, "hello mistral", content[0].Text)
}

func mistralStringPtr(v string) *string {
	return &v
}
