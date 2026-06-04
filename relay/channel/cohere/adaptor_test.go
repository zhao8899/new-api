package cohere

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestBuildsCohereChatPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.ClaudeRequest{
		Model: "command-r-plus",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "text", Text: stringPtr("hello cohere")},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "command-r-plus",
		},
	}

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, info, req)
	require.NoError(t, err)

	cohereReq, ok := got.(*CohereRequest)
	require.True(t, ok)
	require.Equal(t, "command-r-plus", cohereReq.Model)
	require.Equal(t, "hello cohere", cohereReq.Message)
}

func stringPtr(v string) *string {
	return &v
}
