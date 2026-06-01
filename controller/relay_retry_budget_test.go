package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryUsesBudgetForAuthErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := types.NewOpenAIError(errors.New("invalid api key"), types.ErrorCodeBadResponseStatusCode, http.StatusUnauthorized)

	require.False(t, shouldRetry(c, err, 1))
}

func TestShouldRetryAllowsRateLimitFallbackWithinBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := types.NewOpenAIError(errors.New("rate limited"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)

	require.True(t, shouldRetry(c, err, 1))
}

func TestShouldRetryRejectsAfterStreamOutputStarted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	common.SetContextKey(c, constant.ContextKeyIsStream, true)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write([]byte("partial"))

	err := types.NewOpenAIError(errors.New("server error after stream output"), types.ErrorCodeBadResponseStatusCode, http.StatusInternalServerError)

	require.False(t, shouldRetry(c, err, 1))
}

func TestSetRelayRetryHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("use_channel", []string{"101", "202"})

	setRelayRetryHeaders(c, 202, 1)

	require.Equal(t, "1", w.Header().Get("X-NewAPI-Retry-Count"))
	require.Equal(t, "202", w.Header().Get("X-NewAPI-Upstream-Channel"))
	require.Equal(t, "true", w.Header().Get("X-NewAPI-Fallback-Used"))

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	setRelayRetryHeaders(c, 303, 0)
	require.Equal(t, "0", w.Header().Get("X-NewAPI-Retry-Count"))
	require.Equal(t, "303", w.Header().Get("X-NewAPI-Upstream-Channel"))
	require.Equal(t, "false", w.Header().Get("X-NewAPI-Fallback-Used"))
	require.NotEmpty(t, common.RequestIdKey)
}
