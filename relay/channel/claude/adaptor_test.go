package claude

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestBuildsClaudeMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello claude"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-5-sonnet",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	claudeReq, ok := got.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.Equal(t, "claude-3-5-sonnet", claudeReq.Model)
	require.Len(t, claudeReq.Messages, 1)
	require.Equal(t, "user", claudeReq.Messages[0].Role)
	require.Equal(t, "hello claude", claudeReq.Messages[0].GetStringContent())
}

func TestConvertOpenAIResponsesRequestBuildsClaudeMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := dto.OpenAIResponsesRequest{
		Model:        "claude-3-5-sonnet",
		Instructions: []byte(`"system claude"`),
		Input:        []byte(`[{"role":"user","content":[{"type":"input_text","text":"hello claude"}]}]`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-5-sonnet",
		},
	}

	got, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, req)
	require.NoError(t, err)

	claudeReq, ok := got.(*dto.ClaudeRequest)
	require.True(t, ok)
	require.Equal(t, "claude-3-5-sonnet", claudeReq.Model)
	require.Len(t, claudeReq.ParseSystem(), 1)
	require.Equal(t, "system claude", claudeReq.ParseSystem()[0].GetText())
	require.Len(t, claudeReq.Messages, 1)
	require.Equal(t, "hello claude", claudeReq.Messages[0].GetStringContent())
}
