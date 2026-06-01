package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAuditEventTestDB(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&AuditEvent{}))

	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestRecordAuditEventNormalizesAndRedactsDiff(t *testing.T) {
	setupAuditEventTestDB(t)

	event, err := RecordAuditEvent(AuditEventParams{
		ActorID:      10,
		ActorRole:    100,
		Action:       " Channel.Update ",
		ResourceType: " Channel ",
		ResourceID:   "42",
		SourceIP:     "203.0.113.10",
		RequestID:    "req-audit-1",
		Result:       "success",
		DiffRedacted: `{"key":"sk-abcdef1234567890","name":"prod"}`,
		Method:       "PUT",
		Path:         "/api/channel",
		StatusCode:   200,
	})
	require.NoError(t, err)

	require.Equal(t, "channel.update", event.Action)
	require.Equal(t, "channel", event.ResourceType)
	require.Equal(t, "req-audit-1", event.RequestID)
	require.NotContains(t, event.DiffRedacted, "sk-abcdef1234567890")
	require.Contains(t, event.DiffRedacted, "****")
	require.NotZero(t, event.CreatedAt)
}

func TestListAuditEventsByRequestID(t *testing.T) {
	setupAuditEventTestDB(t)

	_, err := RecordAuditEvent(AuditEventParams{
		ActorID:      11,
		Action:       "token.view_key",
		ResourceType: "token",
		ResourceID:   "7",
		RequestID:    "req-audit-2",
		Result:       "success",
	})
	require.NoError(t, err)

	events, err := ListAuditEvents(AuditEventQuery{RequestID: "req-audit-2", Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "token.view_key", events[0].Action)
	require.Equal(t, "7", events[0].ResourceID)
}

func TestListAuditEventsSupportsPaginationAndCount(t *testing.T) {
	setupAuditEventTestDB(t)

	for i := 0; i < 3; i++ {
		_, err := RecordAuditEvent(AuditEventParams{
			ActorID:      12,
			Action:       "channel.update",
			ResourceType: "channel",
			ResourceID:   "42",
			Result:       "success",
		})
		require.NoError(t, err)
	}

	query := AuditEventQuery{
		ActorID:      12,
		Action:       "channel.update",
		ResourceType: "channel",
		StartIdx:     1,
		Limit:        1,
	}
	total, err := CountAuditEvents(query)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)

	events, err := ListAuditEvents(query)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "42", events[0].ResourceID)
}

func TestAuditEventMigrationIncludesCoreColumns(t *testing.T) {
	setupAuditEventTestDB(t)

	require.True(t, LOG_DB.Migrator().HasTable(&AuditEvent{}))
	require.True(t, LOG_DB.Migrator().HasColumn(&AuditEvent{}, "actor_id"))
	require.True(t, LOG_DB.Migrator().HasColumn(&AuditEvent{}, "action"))
	require.True(t, LOG_DB.Migrator().HasColumn(&AuditEvent{}, "resource_type"))
	require.True(t, LOG_DB.Migrator().HasColumn(&AuditEvent{}, "diff_redacted"))
}
