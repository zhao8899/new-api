package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAuditMiddlewareTestDB(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&model.AuditEvent{}))

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestAuditActionRecordsSuccessfulRequest(t *testing.T) {
	setupAuditMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.PUT("/api/channel/:id", func(c *gin.Context) {
		c.Set("id", 100)
		c.Set("role", common.RoleRootUser)
		c.Set(common.RequestIdKey, "req-mw-audit-1")
		c.Set(AuditDiffContextKey, `{"channel_key":"sk-abcdef1234567890"}`)
		c.Next()
	}, AuditAction("channel.update", "channel"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPut, "/api/channel/42", nil)
	req.RemoteAddr = "203.0.113.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	events, err := model.ListAuditEvents(model.AuditEventQuery{RequestID: "req-mw-audit-1", Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, 100, events[0].ActorID)
	require.Equal(t, common.RoleRootUser, events[0].ActorRole)
	require.Equal(t, "channel.update", events[0].Action)
	require.Equal(t, "42", events[0].ResourceID)
	require.Equal(t, "success", events[0].Result)
	require.NotContains(t, events[0].DiffRedacted, "sk-abcdef1234567890")
}

func TestAuditActionRecordsFailureResult(t *testing.T) {
	setupAuditMiddlewareTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.DELETE("/api/token/:id", func(c *gin.Context) {
		c.Set("id", 101)
		c.Set("role", common.RoleAdminUser)
		c.Set(common.RequestIdKey, "req-mw-audit-2")
		c.Next()
	}, AuditAction("token.delete", "token"), func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"success": false})
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/token/9", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	events, err := model.ListAuditEvents(model.AuditEventQuery{RequestID: "req-mw-audit-2", Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "failure", events[0].Result)
	require.Equal(t, http.StatusForbidden, events[0].StatusCode)
}
