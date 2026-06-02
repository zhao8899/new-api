package zhipu

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
			UpstreamModelName: "chatglm_turbo",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	zhReq, ok := got.(*ZhipuRequest)
	require.True(t, ok)
	require.Len(t, zhReq.Prompt, 1)
	require.Equal(t, "user", zhReq.Prompt[0].Role)
	require.Equal(t, "hello zhipu", zhReq.Prompt[0].Content)
	require.Equal(t, 0.99, zhReq.TopP)
}

func TestConvertClaudeRequestBuildsZhipuRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "chatglm_turbo",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: zhipuStringPtr("hello zhipu")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "chatglm_turbo",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	zhReq, ok := got.(*ZhipuRequest)
	require.True(t, ok)
	require.Len(t, zhReq.Prompt, 1)
	require.Equal(t, "user", zhReq.Prompt[0].Role)
	require.Equal(t, "hello zhipu", zhReq.Prompt[0].Content)
}

func zhipuStringPtr(v string) *string {
	return &v
}
