package mokaai

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeRequestReturnsUnsupportedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	got, err := (&Adaptor{}).ConvertClaudeRequest(c, &relaycommon.RelayInfo{}, &dto.ClaudeRequest{})
	require.Nil(t, got)
	require.EqualError(t, err, "mokaai channel: /v1/messages endpoint not supported")
}
