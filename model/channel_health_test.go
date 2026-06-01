package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelHealthTestDB(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&ChannelHealth{}))

	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestRecordChannelHealthSuccessCreatesHealthyState(t *testing.T) {
	setupChannelHealthTestDB(t)

	health, err := RecordChannelHealthSuccess(101, "DeepSeek", "deepseek-chat", 320)
	require.NoError(t, err)

	require.Equal(t, 101, health.ChannelID)
	require.Equal(t, "deepseek", health.Provider)
	require.Equal(t, "deepseek-chat", health.ModelName)
	require.Equal(t, int64(1), health.SuccessCount)
	require.Equal(t, int64(0), health.FailureCount)
	require.Equal(t, 320, health.P95Latency)
	require.Equal(t, ChannelHealthStateHealthy, health.CircuitState)
	require.Equal(t, 100, health.HealthScore)
	require.Zero(t, health.CooldownUntil)
}

func TestRecordChannelHealthFailureMovesRateLimitToCooldown(t *testing.T) {
	setupChannelHealthTestDB(t)

	var health *ChannelHealth
	var err error
	for i := 0; i < 3; i++ {
		health, err = RecordChannelHealthFailure(102, "gemini", "gemini-2.5-pro", "RATE_LIMIT", "quota exceeded", 180)
		require.NoError(t, err)
	}

	require.Equal(t, int64(3), health.FailureCount)
	require.Equal(t, int64(3), health.RateLimitCount)
	require.Equal(t, ChannelHealthStateCooldown, health.CircuitState)
	require.Greater(t, health.CooldownUntil, common.GetTimestamp())
	require.Less(t, health.HealthScore, 100)
}

func TestRecordChannelHealthFailureMovesServerErrorsToOpenCircuit(t *testing.T) {
	setupChannelHealthTestDB(t)

	var health *ChannelHealth
	var err error
	for i := 0; i < 5; i++ {
		health, err = RecordChannelHealthFailure(103, "openai", "gpt-4.1", "SERVER_ERROR", "upstream 500", 900)
		require.NoError(t, err)
	}

	require.Equal(t, int64(5), health.ServerErrorCount)
	require.Equal(t, ChannelHealthStateOpenCircuit, health.CircuitState)
	require.Greater(t, health.CooldownUntil, common.GetTimestamp())
}

func TestRecordChannelHealthFailureDisablesAuthFailuresAndRedactsMessage(t *testing.T) {
	setupChannelHealthTestDB(t)

	var health *ChannelHealth
	var err error
	for i := 0; i < 5; i++ {
		health, err = RecordChannelHealthFailure(104, "anthropic", "claude-sonnet", "AUTH_ERROR", "invalid key sk-abcdef1234567890", 120)
		require.NoError(t, err)
	}

	require.Equal(t, int64(5), health.AuthErrorCount)
	require.Equal(t, ChannelHealthStateDisabled, health.CircuitState)
	require.NotContains(t, health.LastErrorMessage, "sk-abcdef1234567890")
	require.Contains(t, health.LastErrorMessage, "****")
}

func TestGetChannelHealthReturnsMissingWithoutError(t *testing.T) {
	setupChannelHealthTestDB(t)

	health, found, err := GetChannelHealth(999)
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, health)
}
