package xunfei

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestStoresOpenAIRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "spark-max",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: xunfeiStringPtr("hello xunfei")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "spark-max",
		},
	}

	adaptor := &Adaptor{}
	got, err := adaptor.ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	openaiReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Same(t, adaptor.request, openaiReq)
	require.Equal(t, "spark-max", openaiReq.Model)
	require.Len(t, openaiReq.Messages, 1)

	xunfeiReq := requestOpenAI2Xunfei(*openaiReq, "app-id", "generalv3")
	require.Len(t, xunfeiReq.Payload.Message.Text, 1)
	require.Equal(t, "hello xunfei", xunfeiReq.Payload.Message.Text[0].Content)
}

func TestConvertOpenAIResponsesRequestStoresOpenAIRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := dto.OpenAIResponsesRequest{
		Model: "spark-max",
		Input: []byte(`[{"role":"user","content":[{"type":"input_text","text":"hello xunfei"}]}]`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "spark-max",
		},
	}

	adaptor := &Adaptor{}
	got, err := adaptor.ConvertOpenAIResponsesRequest(c, info, req)
	require.NoError(t, err)

	openaiReq, ok := got.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.Same(t, adaptor.request, openaiReq)
	require.Equal(t, "spark-max", openaiReq.Model)

	xunfeiReq := requestOpenAI2Xunfei(*openaiReq, "app-id", "generalv3")
	require.Len(t, xunfeiReq.Payload.Message.Text, 1)
	require.Equal(t, "hello xunfei", xunfeiReq.Payload.Message.Text[0].Content)
}

func xunfeiStringPtr(v string) *string {
	return &v
}
