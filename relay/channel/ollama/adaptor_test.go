package ollama

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestBuildsOllamaChatRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "hello ollama"},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "llama3.1",
		},
	}

	got, err := (&Adaptor{}).ConvertGeminiRequest(c, info, req)
	require.NoError(t, err)

	ollamaReq, ok := got.(*OllamaChatRequest)
	require.True(t, ok)
	require.Equal(t, "llama3.1", ollamaReq.Model)
	require.Len(t, ollamaReq.Messages, 1)
	require.Equal(t, "user", ollamaReq.Messages[0].Role)
	require.Equal(t, "hello ollama", ollamaReq.Messages[0].Content)
}
