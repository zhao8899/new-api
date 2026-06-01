package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProviderRegistryTestDB(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&ProviderRegistry{}))

	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestProviderRegistryInsertNormalizesFields(t *testing.T) {
	setupProviderRegistryTestDB(t)

	registry := &ProviderRegistry{
		Provider:     " DeepSeek ",
		Protocol:     " OPENAI-COMPATIBLE ",
		BaseURL:      " https://api.deepseek.com/ ",
		AuthType:     " Bearer ",
		Enabled:      true,
		HealthStatus: " Healthy ",
	}

	require.NoError(t, registry.Insert())

	got, found, err := GetProviderRegistryByProvider("deepseek")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "deepseek", got.Provider)
	require.Equal(t, ModelProtocolOpenAICompatible, got.Protocol)
	require.Equal(t, "https://api.deepseek.com", got.BaseURL)
	require.Equal(t, ProviderAuthBearer, got.AuthType)
	require.Equal(t, ProviderHealthHealthy, got.HealthStatus)
}

func TestGetDefaultProviderMetadataByChannelType(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		provider    string
		protocol    string
		task        bool
	}{
		{
			name:        "openai",
			channelType: constant.ChannelTypeOpenAI,
			provider:    "openai",
			protocol:    ModelProtocolOpenAI,
		},
		{
			name:        "claude",
			channelType: constant.ChannelTypeAnthropic,
			provider:    "anthropic",
			protocol:    ModelProtocolClaude,
		},
		{
			name:        "gemini",
			channelType: constant.ChannelTypeGemini,
			provider:    "gemini",
			protocol:    ModelProtocolGemini,
		},
		{
			name:        "deepseek",
			channelType: constant.ChannelTypeDeepSeek,
			provider:    "deepseek",
			protocol:    ModelProtocolOpenAICompatible,
		},
		{
			name:        "qwen dashscope",
			channelType: constant.ChannelTypeAli,
			provider:    "qwen",
			protocol:    ModelProtocolDashScope,
		},
		{
			name:        "local ollama",
			channelType: constant.ChannelTypeOllama,
			provider:    "ollama",
			protocol:    ModelProtocolLocal,
		},
		{
			name:        "task suno",
			channelType: constant.ChannelTypeSunoAPI,
			provider:    "suno",
			protocol:    ModelProtocolTask,
			task:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDefaultProviderMetadataByChannelType(tt.channelType)
			require.Equal(t, tt.provider, got.Provider)
			require.Equal(t, tt.protocol, got.Protocol)
			require.Equal(t, tt.task, got.Task)
			require.Equal(t, tt.channelType, got.ChannelType)
		})
	}
}
