package tencent

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestBuildsTencentPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(string(constant.ContextKeyChannelKey), "123|secret-id|secret-key")

	req := &dto.ClaudeRequest{
		Model: "hunyuan-turbo",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: stringPtrTencent("hello tencent")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "hunyuan-turbo",
		},
	}

	adaptor := &Adaptor{}
	adaptor.Init(info)
	got, err := adaptor.ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	tencentReq, ok := got.(*TencentChatRequest)
	require.True(t, ok)
	require.NotEmpty(t, adaptor.Sign)
	require.Equal(t, int64(123), adaptor.AppID)
	require.Equal(t, "hunyuan-turbo", *tencentReq.Model)
	require.Len(t, tencentReq.Messages, 1)
	require.Equal(t, "hello tencent", tencentReq.Messages[0].Content)
}

func TestConvertOpenAIResponsesRequestBuildsTencentPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(string(constant.ContextKeyChannelKey), "123|secret-id|secret-key")

	req := dto.OpenAIResponsesRequest{
		Model: "hunyuan-turbo",
		Input: []byte(`[{"role":"user","content":[{"type":"input_text","text":"hello tencent"}]}]`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "hunyuan-turbo",
		},
	}

	adaptor := &Adaptor{}
	adaptor.Init(info)
	got, err := adaptor.ConvertOpenAIResponsesRequest(c, info, req)
	require.NoError(t, err)

	tencentReq, ok := got.(*TencentChatRequest)
	require.True(t, ok)
	require.NotEmpty(t, adaptor.Sign)
	require.Equal(t, int64(123), adaptor.AppID)
	require.Len(t, tencentReq.Messages, 1)
	require.Equal(t, "hello tencent", tencentReq.Messages[0].Content)
}

func stringPtrTencent(v string) *string {
	return &v
}
