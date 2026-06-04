package baidu

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestBuildsBaiduPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "ERNIE-Bot",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: baiduStringPtr("hello baidu")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "ERNIE-Bot",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	baiduReq, ok := got.(*BaiduChatRequest)
	require.True(t, ok)
	require.Len(t, baiduReq.Messages, 1)
	require.Equal(t, "user", baiduReq.Messages[0].Role)
	require.Equal(t, "hello baidu", baiduReq.Messages[0].Content)
}

func baiduStringPtr(v string) *string {
	return &v
}
