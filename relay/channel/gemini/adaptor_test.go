package gemini

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestBuildsGeminiRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	req := dto.OpenAIResponsesRequest{
		Model:        "gemini-2.5-flash",
		Instructions: []byte(`"system gemini"`),
		Input:        []byte(`[{"role":"user","content":[{"type":"input_text","text":"hello gemini"}]}]`),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-2.5-flash",
		},
	}

	got, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, req)
	require.NoError(t, err)

	geminiReq, ok := got.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.NotNil(t, geminiReq.SystemInstructions)
	require.Len(t, geminiReq.SystemInstructions.Parts, 1)
	require.Equal(t, "system gemini", geminiReq.SystemInstructions.Parts[0].Text)
	require.Len(t, geminiReq.Contents, 1)
	require.Equal(t, "user", geminiReq.Contents[0].Role)
	require.Len(t, geminiReq.Contents[0].Parts, 1)
	require.Equal(t, "hello gemini", geminiReq.Contents[0].Parts[0].Text)
}
