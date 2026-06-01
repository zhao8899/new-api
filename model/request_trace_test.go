package model

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRequestTraceTestDB(t *testing.T) {
	t.Helper()

	originalDB := DB
	originalLogDB := LOG_DB
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	LOG_DB = db
	require.NoError(t, db.AutoMigrate(&RequestTrace{}))

	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestRecordRequestTraceCreatesAndNormalizesTrace(t *testing.T) {
	setupRequestTraceTestDB(t)

	trace, err := RecordRequestTrace(RequestTraceParams{
		RequestID:          "req-001",
		TraceID:            "trace-001",
		UserID:             10,
		TokenID:            20,
		Group:              "vip",
		ExternalModel:      "chatgpt-4o",
		InternalModel:      "gpt-4o",
		Provider:           " OpenAI ",
		ChannelID:          30,
		UpstreamModel:      "gpt-4o-2024-11-20",
		RetryCount:         1,
		FallbackUsed:       true,
		StatusCode:         200,
		UpstreamStatusCode: 200,
		LatencyMS:          530,
		PromptTokens:       11,
		CompletionTokens:   13,
		EstimatedCost:      0.12,
		ActualCost:         0.10,
	})
	require.NoError(t, err)

	require.Equal(t, "req-001", trace.RequestID)
	require.Equal(t, "openai", trace.Provider)
	require.Equal(t, 24, trace.TotalTokens)
	require.Equal(t, 530, trace.LatencyMS)
	require.True(t, trace.FallbackUsed)
	require.NotZero(t, trace.CreatedAt)
	require.NotZero(t, trace.UpdatedAt)
}

func TestRecordRequestTraceUpsertsByRequestIDAndRedactsError(t *testing.T) {
	setupRequestTraceTestDB(t)

	_, err := RecordRequestTrace(RequestTraceParams{
		RequestID:  "req-002",
		TraceID:    "trace-002",
		Provider:   "anthropic",
		StatusCode: 500,
	})
	require.NoError(t, err)

	trace, err := RecordRequestTrace(RequestTraceParams{
		RequestID:            "req-002",
		TraceID:              "trace-002",
		Provider:             "anthropic",
		StatusCode:           502,
		UpstreamStatusCode:   500,
		ErrorType:            "SERVER_ERROR",
		ErrorMessageRedacted: "upstream failed with key sk-abcdef1234567890",
		PromptTokens:         5,
		CompletionTokens:     7,
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, LOG_DB.Model(&RequestTrace{}).Where("request_id = ?", "req-002").Count(&count).Error)
	require.Equal(t, int64(1), count)
	require.Equal(t, 502, trace.StatusCode)
	require.Equal(t, "SERVER_ERROR", trace.ErrorType)
	require.Equal(t, 12, trace.TotalTokens)
	require.NotContains(t, trace.ErrorMessageRedacted, "sk-abcdef1234567890")
	require.Contains(t, trace.ErrorMessageRedacted, "****")
}

func TestRecordRequestTraceRejectsMissingRequestID(t *testing.T) {
	setupRequestTraceTestDB(t)

	trace, err := RecordRequestTrace(RequestTraceParams{Provider: "openai"})

	require.Error(t, err)
	require.Nil(t, trace)
}

func TestRequestTraceMigrationIncludesCoreColumns(t *testing.T) {
	setupRequestTraceTestDB(t)

	require.True(t, LOG_DB.Migrator().HasTable(&RequestTrace{}))
	require.True(t, LOG_DB.Migrator().HasColumn(&RequestTrace{}, "request_id"))
	require.True(t, LOG_DB.Migrator().HasColumn(&RequestTrace{}, "provider"))
	require.True(t, LOG_DB.Migrator().HasColumn(&RequestTrace{}, "channel_id"))
	require.True(t, LOG_DB.Migrator().HasColumn(&RequestTrace{}, "error_type"))
}

func TestRecordConsumeLogCreatesRequestTrace(t *testing.T) {
	setupRequestTraceLogTestDB(t)

	c := newRequestTraceGinContext("req-consume")
	c.Set("username", "alice")
	c.Set("channel_type", constant.ChannelTypeOpenAI)
	common.LogConsumeEnabled = true

	RecordConsumeLog(c, 7, RecordConsumeLogParams{
		ChannelId:        8,
		PromptTokens:     10,
		CompletionTokens: 20,
		ModelName:        "gpt-4.1",
		TokenName:        "prod-token",
		Quota:            int(common.QuotaPerUnit),
		TokenId:          9,
		UseTimeSeconds:   2,
		Group:            "vip",
	})

	trace, found, err := GetRequestTraceByRequestID("req-consume")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 7, trace.UserID)
	require.Equal(t, 9, trace.TokenID)
	require.Equal(t, "openai", trace.Provider)
	require.Equal(t, 30, trace.TotalTokens)
	require.Equal(t, 2000, trace.LatencyMS)
	require.Equal(t, 1.0, trace.ActualCost)
}

func TestRecordErrorLogCreatesRedactedRequestTrace(t *testing.T) {
	setupRequestTraceLogTestDB(t)

	c := newRequestTraceGinContext("req-error")
	c.Set("username", "alice")
	c.Set("channel_type", constant.ChannelTypeAnthropic)

	RecordErrorLog(c, 7, 8, "claude-sonnet", "prod-token", "failed with sk-abcdef1234567890", 9, 3, true, "vip", map[string]interface{}{
		"status_code": 502,
		"error_type":  "SERVER_ERROR",
	})

	trace, found, err := GetRequestTraceByRequestID("req-error")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "anthropic", trace.Provider)
	require.Equal(t, 502, trace.StatusCode)
	require.Equal(t, "SERVER_ERROR", trace.ErrorType)
	require.Equal(t, 3000, trace.LatencyMS)
	require.NotContains(t, trace.ErrorMessageRedacted, "sk-abcdef1234567890")
	require.Contains(t, trace.ErrorMessageRedacted, "****")
}

func setupRequestTraceLogTestDB(t *testing.T) {
	t.Helper()

	originalLogConsumeEnabled := common.LogConsumeEnabled
	setupRequestTraceTestDB(t)
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}, &RequestTrace{}))

	t.Cleanup(func() {
		common.LogConsumeEnabled = originalLogConsumeEnabled
	})
}

func newRequestTraceGinContext(requestID string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(common.RequestIdKey, requestID)
	return c
}
