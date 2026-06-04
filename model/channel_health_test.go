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

func TestIsChannelHealthRoutableUsesCircuitState(t *testing.T) {
	setupChannelHealthTestDB(t)
	t.Setenv("NEW_API_CHANNEL_CIRCUIT_MODE", "enforce")

	ok, err := IsChannelHealthRoutable(501)
	require.NoError(t, err)
	require.True(t, ok)

	_, err = RecordChannelHealthFailure(501, "openai", "gpt-4.1", "RATE_LIMIT", "quota exceeded", 120)
	require.NoError(t, err)
	_, err = RecordChannelHealthFailure(501, "openai", "gpt-4.1", "RATE_LIMIT", "quota exceeded", 120)
	require.NoError(t, err)
	_, err = RecordChannelHealthFailure(501, "openai", "gpt-4.1", "RATE_LIMIT", "quota exceeded", 120)
	require.NoError(t, err)

	ok, err = IsChannelHealthRoutable(501)
	require.NoError(t, err)
	require.False(t, ok)

	for i := 0; i < 5; i++ {
		_, err = RecordChannelHealthFailure(502, "anthropic", "claude-sonnet", "AUTH_ERROR", "invalid key", 120)
		require.NoError(t, err)
	}
	ok, err = IsChannelHealthRoutable(502)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestIsChannelHealthRoutableDefaultsToObserveMode(t *testing.T) {
	setupChannelHealthTestDB(t)

	for i := 0; i < 3; i++ {
		_, err := RecordChannelHealthFailure(504, "gemini", "gemini-2.5-pro", "RATE_LIMIT", "quota exceeded", 120)
		require.NoError(t, err)
	}

	health, found, err := GetChannelHealth(504)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, ChannelHealthStateCooldown, health.CircuitState)

	ok, err := IsChannelHealthRoutable(504)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestChannelHealthCircuitModeUsesOptionBeforeEnv(t *testing.T) {
	setupChannelHealthTestDB(t)
	t.Setenv("NEW_API_CHANNEL_CIRCUIT_MODE", "enforce")

	originalOptionMap := common.OptionMap
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{ChannelHealthCircuitModeOption: "observe"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	require.Equal(t, channelHealthCircuitModeObserve, ChannelHealthCircuitMode())
}

func TestListChannelHealthFiltersAndCounts(t *testing.T) {
	setupChannelHealthTestDB(t)

	_, err := RecordChannelHealthFailure(701, "gemini", "gemini-2.5-pro", "RATE_LIMIT", "quota exceeded", 100)
	require.NoError(t, err)
	_, err = RecordChannelHealthFailure(701, "gemini", "gemini-2.5-pro", "RATE_LIMIT", "quota exceeded", 100)
	require.NoError(t, err)
	_, err = RecordChannelHealthFailure(701, "gemini", "gemini-2.5-pro", "RATE_LIMIT", "quota exceeded", 100)
	require.NoError(t, err)
	_, err = RecordChannelHealthSuccess(702, "openai", "gpt-4.1", 80)
	require.NoError(t, err)

	query := ChannelHealthQuery{
		Provider:      "gemini",
		CircuitState:  ChannelHealthStateCooldown,
		LastErrorType: "rate_limit",
		StartIdx:      0,
		Limit:         10,
	}
	total, err := CountChannelHealth(query)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	items, err := ListChannelHealth(query)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, 701, items[0].ChannelID)
	require.Equal(t, ChannelHealthStateCooldown, items[0].CircuitState)
}

func TestIsChannelHealthRoutableAllowsExpiredCooldownProbe(t *testing.T) {
	setupChannelHealthTestDB(t)
	t.Setenv("NEW_API_CHANNEL_CIRCUIT_MODE", "enforce")

	health := &ChannelHealth{
		ChannelID:     503,
		Provider:      "gemini",
		ModelName:     "gemini-2.5-pro",
		HealthScore:   50,
		CircuitState:  ChannelHealthStateCooldown,
		CooldownUntil: common.GetTimestamp() - 1,
		CreatedTime:   common.GetTimestamp(),
		UpdatedTime:   common.GetTimestamp(),
	}
	require.NoError(t, DB.Select("*").Create(health).Error)

	ok, err := IsChannelHealthRoutable(503)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestGetChannelSkipsUnroutableChannelHealth(t *testing.T) {
	setupChannelHealthTestDB(t)
	t.Setenv("NEW_API_CHANNEL_CIRCUIT_MODE", "enforce")
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = false
	})
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &ChannelHealth{}))

	priority := int64(10)
	weight := uint(100)
	require.NoError(t, DB.Create(&Channel{Id: 601, Type: 1, Key: "sk-a", Status: common.ChannelStatusEnabled, Name: "cooldown", Priority: &priority, Weight: &weight}).Error)
	require.NoError(t, DB.Create(&Ability{Group: "default", Model: "gpt-4.1", ChannelId: 601, Enabled: true, Priority: &priority, Weight: weight}).Error)

	for i := 0; i < 3; i++ {
		_, err := RecordChannelHealthFailure(601, "openai", "gpt-4.1", "RATE_LIMIT", "quota exceeded", 120)
		require.NoError(t, err)
	}

	channel, err := GetChannel("default", "gpt-4.1", 0)
	require.NoError(t, err)
	require.Nil(t, channel)
}
