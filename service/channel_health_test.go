package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupServiceChannelHealthTestDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.ChannelHealth{}))

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestRecordChannelHealthFromRelayErrorClassifiesRateLimit(t *testing.T) {
	setupServiceChannelHealthTestDB(t)

	relayErr := types.NewErrorWithStatusCode(errors.New("quota exceeded"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
	for i := 0; i < 3; i++ {
		require.NoError(t, RecordChannelHealthFromRelayError(201, "gemini", "gemini-2.5-pro", relayErr, 260))
	}

	health, found, err := model.GetChannelHealth(201)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, int64(3), health.RateLimitCount)
	require.Equal(t, model.ChannelHealthStateCooldown, health.CircuitState)
}

func TestRecordChannelHealthFromRelayErrorSkipsNilError(t *testing.T) {
	setupServiceChannelHealthTestDB(t)

	require.NoError(t, RecordChannelHealthFromRelayError(202, "openai", "gpt-4.1", nil, 120))

	health, found, err := model.GetChannelHealth(202)
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, health)
}

func TestRecordChannelHealthSuccessRecordsProviderAndModel(t *testing.T) {
	setupServiceChannelHealthTestDB(t)

	require.NoError(t, RecordChannelHealthSuccess(203, "deepseek", "deepseek-chat", 180))

	health, found, err := model.GetChannelHealth(203)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "deepseek", health.Provider)
	require.Equal(t, "deepseek-chat", health.ModelName)
	require.Equal(t, int64(1), health.SuccessCount)
}

func TestRecordChannelHealthSuccessFromContext(t *testing.T) {
	setupServiceChannelHealthTestDB(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("channel_id", 204)
	c.Set("channel_type", constant.ChannelTypeDeepSeek)
	c.Set("original_model", "deepseek-chat")
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now().Add(-250*time.Millisecond))

	require.NoError(t, RecordChannelHealthSuccessFromContext(c))

	health, found, err := model.GetChannelHealth(204)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "deepseek", health.Provider)
	require.Equal(t, "deepseek-chat", health.ModelName)
	require.Equal(t, int64(1), health.SuccessCount)
	require.GreaterOrEqual(t, health.P95Latency, 200)
}
