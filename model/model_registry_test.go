package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelRegistryTestDB(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&ModelRegistry{}))

	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestModelRegistryInsertNormalizesFields(t *testing.T) {
	setupModelRegistryTestDB(t)

	registry := &ModelRegistry{
		ExternalModel:   " qwen-max ",
		Provider:        " QWEN ",
		Protocol:        " OPENAI-COMPATIBLE ",
		Capabilities:    "chat, tool, chat, CODE ",
		ContextWindow:   128000,
		MaxOutputTokens: 8192,
		Enabled:         true,
	}

	require.NoError(t, registry.Insert())

	got, found, err := GetModelRegistryByExternalModel("qwen-max")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "qwen-max", got.ExternalModel)
	require.Equal(t, "qwen", got.Provider)
	require.Equal(t, "qwen-max", got.UpstreamModel)
	require.Equal(t, ModelProtocolOpenAICompatible, got.Protocol)
	require.Equal(t, "chat,tool,code", got.Capabilities)
	require.Equal(t, 128000, got.ContextWindow)
	require.Equal(t, 8192, got.MaxOutputTokens)
}

func TestGetModelRegistryByExternalModelIgnoresDisabled(t *testing.T) {
	setupModelRegistryTestDB(t)

	registry := &ModelRegistry{
		ExternalModel: "disabled-model",
		Provider:      "openai",
		UpstreamModel: "gpt-disabled",
		Protocol:      ModelProtocolOpenAI,
		Enabled:       false,
	}
	require.NoError(t, registry.Insert())

	got, found, err := GetModelRegistryByExternalModel("disabled-model")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, got)
}

func TestGetModelRegistryByExternalModelUsesHighestPriorityAcrossProviders(t *testing.T) {
	setupModelRegistryTestDB(t)

	lowPriority := &ModelRegistry{
		ExternalModel: "gpt-commercial",
		Provider:      "openai",
		UpstreamModel: "gpt-low",
		Protocol:      ModelProtocolOpenAI,
		Enabled:       true,
		Priority:      1,
	}
	highPriority := &ModelRegistry{
		ExternalModel: "gpt-commercial",
		Provider:      "azure-openai",
		UpstreamModel: "gpt-high",
		Protocol:      ModelProtocolAzureOpenAI,
		Enabled:       true,
		Priority:      10,
	}

	require.NoError(t, lowPriority.Insert())
	require.NoError(t, highPriority.Insert())

	got, found, err := GetModelRegistryByExternalModel("gpt-commercial")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "azure-openai", got.Provider)
	require.Equal(t, "gpt-high", got.UpstreamModel)
}

func TestModelRegistryMigrationIncludesRegistry(t *testing.T) {
	setupModelRegistryTestDB(t)

	require.True(t, DB.Migrator().HasTable(&ModelRegistry{}))
	require.True(t, DB.Migrator().HasColumn(&ModelRegistry{}, "external_model"))
	require.True(t, DB.Migrator().HasColumn(&ModelRegistry{}, "provider"))
	require.True(t, DB.Migrator().HasColumn(&ModelRegistry{}, "upstream_model"))
	require.True(t, DB.Migrator().HasColumn(&ModelRegistry{}, "protocol"))
}
